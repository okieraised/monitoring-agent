package constants

import "mime"

const (
	APIFieldRequestID = "request_id"
)

const (
	ContentTypeOctetStream = "application/octet-stream"
	ContentTypeJSON        = "application/json"
	ContentTypeProblemJSON = "application/problem+json"
	ContentTypeLDJSON      = "application/ld+json"
	ContentTypeNDJSON      = "application/x-ndjson"
	ContentTypeForm        = "application/x-www-form-urlencoded"
	ContentTypeMultipart   = "multipart/form-data"
	ContentTypeHTML        = "text/html; charset=utf-8"
	ContentTypeTextUTF8    = "text/plain; charset=utf-8"
	ContentTypeText        = "text/plain"
	ContentTypeXML         = "application/xml" // prefer over text/xml
	ContentTypePDF         = "application/pdf"
	ContentTypeZip         = "application/zip"
	ContentTypeGZip        = "application/gzip"
	ContentTypeTar         = "application/x-tar"
	ContentTypeSevenZip    = "application/x-7z-compressed"
)

const (
	ContentTypeJPEG = "image/jpeg"
	ContentTypePNG  = "image/png"
	ContentTypeGIF  = "image/gif"
	ContentTypeWEBP = "image/webp"
	ContentTypeSVG  = "image/svg+xml"
	ContentTypeAVIF = "image/avif"

	ContentTypeMP4  = "video/mp4"
	ContentTypeMPEG = "video/mpeg"
	ContentTypeWEBM = "video/webm"
)

const ContentTypeImageFmt = "image/%s"

const (
	ContentTypeRAR  = "application/x-rar-compressed"
	ContentTypeJS   = "application/javascript"
	ContentTypeCSS  = "text/css"
	ContentTypeWASM = "application/wasm"
)

const (
	HeaderAccept                  = "Accept"
	HeaderAcceptEncoding          = "Accept-Encoding"
	HeaderAcceptLanguage          = "Accept-Language"
	HeaderAuthorization           = "Authorization"
	HeaderCacheControl            = "Cache-Control"
	HeaderConnection              = "Connection"
	HeaderContentDisposition      = "Content-Disposition"
	HeaderContentEncoding         = "Content-Encoding"
	HeaderContentLanguage         = "Content-Language"
	HeaderContentLength           = "Content-Length"
	HeaderContentLocation         = "Content-Location"
	HeaderContentMD5              = "Content-MD5" // non-standard but seen in the wild
	HeaderContentType             = "Content-Type"
	HeaderContentDigest           = "Content-Digest"
	HeaderContentTransferEncoding = "Content-Transfer-Encoding"
	HeaderETag                    = "ETag"
	HeaderExpires                 = "Expires"
	HeaderHost                    = "Host"
	HeaderIfMatch                 = "If-Match"
	HeaderIfNoneMatch             = "If-None-Match"
	HeaderIfModifiedSince         = "If-Modified-Since"
	HeaderIfUnmodifiedSince       = "If-Unmodified-Since"
	HeaderLastModified            = "Last-Modified"
	HeaderLink                    = "Link"
	HeaderLocation                = "Location"
	HeaderOrigin                  = "Origin"
	HeaderPragma                  = "Pragma"
	HeaderRange                   = "Range"
	HeaderRetryAfter              = "Retry-After"
	HeaderServer                  = "Server"
	HeaderTrailer                 = "Trailer"
	HeaderTransferEncoding        = "Transfer-Encoding"
	HeaderUpgrade                 = "Upgrade"
	HeaderUserAgent               = "User-Agent"
	HeaderVary                    = "Vary"
	HeaderVia                     = "Via"
	HeaderWWWAuthenticate         = "WWW-Authenticate"

	// Cookies
	HeaderCookie     = "Cookie"
	HeaderSetCookie  = "Set-Cookie"
	HeaderProxyAuth  = "Proxy-Authorization"
	HeaderProxyAuthN = "Proxy-Authenticate"

	// CORS / Proxy-related
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	HeaderAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"
	HeaderAccessControlRequestHeaders   = "Access-Control-Request-Headers"
	HeaderAccessControlRequestMethod    = "Access-Control-Request-Method"

	HeaderXForwardedFor    = "X-Forwarded-For"
	HeaderXForwardedHost   = "X-Forwarded-Host"
	HeaderXForwardedProto  = "X-Forwarded-Proto"
	HeaderXForwardedScheme = "X-Forwarded-Scheme"

	// Common X- headers
	HeaderXAPIKey             = "X-API-Key" // #nosec G101
	HeaderXRequestID          = "X-Request-ID"
	HeaderXRequestedWith      = "X-Requested-With"
	HeaderXRateLimitLimit     = "X-RateLimit-Limit"
	HeaderXRateLimitRemaining = "X-RateLimit-Remaining"
	HeaderXRateLimitReset     = "X-RateLimit-Reset"
	HeaderXRateLimitWindow    = "X-RateLimit-Window"
	HeaderXRateLimitCount     = "X-RateLimit-Count"
	HeaderXContentTypeOptions = "X-Content-Type-Options" // nosniff
	HeaderXFrameOptions       = "X-Frame-Options"
	HeaderXXSSProtection      = "X-XSS-Protection" // legacy
	HeaderXLicenseChecksum    = "X-License-Checksum"
	HeaderXMachineChecksum    = "X-Machine-Checksum"
)

func init() {
	_ = mime.AddExtensionType(".svg", ContentTypeSVG)
	_ = mime.AddExtensionType(".json", ContentTypeJSON)
	_ = mime.AddExtensionType(".jsonl", ContentTypeNDJSON)
	_ = mime.AddExtensionType(".ndjson", ContentTypeNDJSON)
	_ = mime.AddExtensionType(".wasm", ContentTypeWASM)
	_ = mime.AddExtensionType(".webp", ContentTypeWEBP)
	_ = mime.AddExtensionType(".avif", ContentTypeAVIF)
	_ = mime.AddExtensionType(".mjs", ContentTypeJS)
	_ = mime.AddExtensionType(".md", "text/markdown; charset=utf-8")
	_ = mime.AddExtensionType(".yml", "application/yaml")
	_ = mime.AddExtensionType(".yaml", "application/yaml")
}
