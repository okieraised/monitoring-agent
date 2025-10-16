package mcap_schema_resolver

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/okieraised/monitoring-agent/internal/constants"
	"github.com/okieraised/monitoring-agent/internal/utilities"
)

type CentralSchemaResolver struct {
	ResolvedMap map[string]string
	SearchPaths []string
}

// NewCentralSchemaResolver constructs an empty resolver.
func NewCentralSchemaResolver() *CentralSchemaResolver {
	return &CentralSchemaResolver{
		ResolvedMap: make(map[string]string),
		SearchPaths: make([]string, 0),
	}
}

var (
	globalResolver *CentralSchemaResolver
	globalOnce     sync.Once
	globalInitErr  error
)

// Global returns a singleton CentralSchemaResolver initialized from AmentPrefixPath.
// Returns an error if initialization fails or no .msg files were found.
func Global() (*CentralSchemaResolver, error) {
	globalOnce.Do(func() {
		r := NewCentralSchemaResolver()
		if err := r.RegisterRos2FromEnv(); err != nil {
			globalInitErr = err
			return
		}
		if len(r.ResolvedMap) == 0 {
			globalInitErr = fmt.Errorf("no .msg files found under AMENT_PREFIX_PATH")
			return
		}
		globalResolver = r
	})
	if globalInitErr != nil {
		return nil, globalInitErr
	}
	return globalResolver, nil
}

// RegisterMsgDir scans a directory for .msg files and registers them under "package/msg/Name".
func (r *CentralSchemaResolver) RegisterMsgDir(pkg string, path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if ext := filepath.Ext(e.Name()); ext == ".msg" {
			msgName := strings.TrimSuffix(e.Name(), ext)
			key := fmt.Sprintf("%s/msg/%s", pkg, msgName)
			r.ResolvedMap[key] = filepath.Join(path, e.Name())
		}
	}
	return nil
}

// RegisterRos2FromEnv reads AmentPrefixPath and registers standard ros2 msg dirs.
func (r *CentralSchemaResolver) RegisterRos2FromEnv() error {
	osEnv, ok := os.LookupEnv(constants.AmentPrefixPath)
	if !ok || strings.TrimSpace(osEnv) == "" {
		return fmt.Errorf("%s is not set", constants.AmentPrefixPath)
	}
	// Split by OS path list separator (':' on Unix)
	parts := strings.Split(osEnv, string(os.PathListSeparator))
	return r.RegisterRos2StandardPathsAny(parts)
}

// Resolve reads the .msg file for typename and returns trimmed contents.
func (r *CentralSchemaResolver) Resolve(typename string) (string, error) {
	path, ok := r.ResolvedMap[typename]
	if !ok {
		return "", fmt.Errorf("definition not found for: %s", typename)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// RegisterRos2StandardPaths accepts a colon-separated list
func (r *CentralSchemaResolver) RegisterRos2StandardPaths(amentPrefixes string) error {
	for _, prefix := range strings.Split(amentPrefixes, ":") {
		if strings.TrimSpace(prefix) == "" {
			continue
		}
		shareDir := filepath.Join(prefix, "share")
		if stat, err := os.Stat(shareDir); err != nil || !stat.IsDir() {
			continue
		}
		entries, err := os.ReadDir(shareDir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			packageDir := filepath.Join(shareDir, entry.Name())
			msgDir := filepath.Join(packageDir, "msg")
			if stat, err := os.Stat(msgDir); err == nil && stat.IsDir() {
				pkgName := filepath.Base(packageDir)
				if err := r.RegisterMsgDir(pkgName, msgDir); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// RegisterRos2StandardPathsAny accepts an iterable of prefixes (paths) and registers packages/msg in each.
func (r *CentralSchemaResolver) RegisterRos2StandardPathsAny(prefixes []string) error {
	// copy and sort for deterministic behavior
	p := make([]string, 0, len(prefixes))
	for _, pre := range prefixes {
		if strings.TrimSpace(pre) != "" {
			p = append(p, pre)
		}
	}
	sort.Strings(p)

	for _, prefix := range p {
		shareDir := filepath.Join(prefix, "share")
		stat, err := os.Stat(shareDir)
		if err != nil || !stat.IsDir() {
			continue
		}

		entries, err := os.ReadDir(shareDir)
		if err != nil {
			return err
		}
		packages := make([]string, 0, len(entries))
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			packages = append(packages, filepath.Join(shareDir, e.Name()))
		}
		sort.Strings(packages)

		for _, packageDir := range packages {
			msgDir := filepath.Join(packageDir, "msg")
			if stat, err := os.Stat(msgDir); err == nil && stat.IsDir() {
				packageName := filepath.Base(packageDir)
				if err := r.RegisterMsgDir(packageName, msgDir); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Flatten returns a flattened string containing the root definition and all dependencies (separated by blank lines).
func (r *CentralSchemaResolver) Flatten(rootType string) (string, error) {
	visited := make(map[string]struct{})
	flat := make([]string, 0)

	definition, err := r.Resolve(rootType)
	if err != nil {
		return "", err
	}
	flat = append(flat, definition)
	if err := r.flattenInner(rootType, definition, visited, &flat); err != nil {
		return "", err
	}
	return strings.Join(flat, "\n\n"), nil
}

// isBuiltinType checks whether the raw type is a ROS builtin (including bounded strings like string<=N).
func (r *CentralSchemaResolver) isBuiltinType(raw string) bool {
	base := utilities.StripArraySuffix(raw)
	switch base {
	case "bool", "byte", "char",
		"int8", "uint8", "int16", "uint16",
		"int32", "uint32", "int64", "uint64",
		"float32", "float64",
		"string", "wstring":
		return true
	}

	// bounded strings: string<=N or wstring<=N
	if strings.HasPrefix(base, "string") {
		rest := base[len("string"):]
		if rest == "" {
			return true
		}
		if strings.HasPrefix(rest, "<=") && utilities.AllDigits(rest[2:]) {
			return true
		}
	}
	if strings.HasPrefix(base, "wstring") {
		rest := base[len("wstring"):]
		if rest == "" {
			return true
		}
		if strings.HasPrefix(rest, "<=") && utilities.AllDigits(rest[2:]) {
			return true
		}
	}

	return false
}

// resolveCustomType returns (resolvedType, true) if this raw type should be resolved to a custom message type.
// For builtins returns ("", false).
func (r *CentralSchemaResolver) resolveCustomType(raw string, currentPackage string) (string, bool) {
	if r.isBuiltinType(raw) {
		return "", false
	}
	base := utilities.StripArraySuffix(raw)
	if strings.Contains(base, "/") {
		if strings.Contains(base, "/msg/") {
			return base, true
		}
		segments := strings.Split(base, "/")
		if len(segments) == 2 {
			return fmt.Sprintf("%s/msg/%s", segments[0], segments[1]), true
		}
		return "", false
	}
	return fmt.Sprintf("%s/msg/%s", currentPackage, base), true
}

func (r *CentralSchemaResolver) flattenInner(currentType string, definition string, visited map[string]struct{}, flat *[]string) error {
	visited[currentType] = struct{}{}
	currentPackage := ""
	if idx := strings.Index(currentType, "/"); idx >= 0 {
		currentPackage = currentType[:idx]
	}

	for _, line := range strings.Split(definition, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		typename := parts[0]
		if nestedType, ok := r.resolveCustomType(typename, currentPackage); ok {
			typenameOut := strings.ReplaceAll(nestedType, "/msg/", "/")
			if _, seen := visited[nestedType]; !seen {
				nestedDef, err := r.Resolve(nestedType)
				if err != nil {
					return err
				}
				entry := fmt.Sprintf(
					"================================================================================\nMSG: %s\n%s",
					typenameOut,
					strings.TrimSpace(nestedDef),
				)
				*flat = append(*flat, entry)
				if err := r.flattenInner(nestedType, nestedDef, visited, flat); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// GenerateAllRaw returns a map of typename -> raw message text for every registered message.
func (r *CentralSchemaResolver) GenerateAllRaw() (map[string]string, error) {
	out := make(map[string]string, len(r.ResolvedMap))
	keys := make([]string, 0, len(r.ResolvedMap))
	for k := range r.ResolvedMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v, err := r.Resolve(k)
		if err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, nil
}

// GenerateAllFlattened returns a map of typename -> flattened text (root + deps).
func (r *CentralSchemaResolver) GenerateAllFlattened() (map[string]string, error) {
	out := make(map[string]string, len(r.ResolvedMap))
	keys := make([]string, 0, len(r.ResolvedMap))
	for k := range r.ResolvedMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("key: %s\n", k)
		f, err := r.Flatten(k)
		if err != nil {
			return nil, err
		}
		out[k] = f
	}
	return out, nil
}
