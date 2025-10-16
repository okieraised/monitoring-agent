package utilities

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"strings"
)

func EncodeROSImageToBytes(
	width, height int,
	encoding string,
	step int,
	isBigEndian bool,
	data []byte,
	format string,
	quality int,
) ([]byte, error) {
	enc := strings.ToLower(strings.TrimSpace(encoding))
	fmtFmt := strings.ToLower(strings.TrimSpace(format))
	if fmtFmt == "jpg" {
		fmtFmt = "jpeg"
	}
	if fmtFmt != "png" && fmtFmt != "jpeg" {
		return nil, fmt.Errorf("unsupported output format %q (want png or jpeg)", format)
	}
	if quality < 1 || quality > 100 {
		quality = 90
	}

	inferStep := func(ch, bytesPerChan int) int {
		if step > 0 {
			return step
		}
		return width * ch * bytesPerChan
	}

	var img image.Image

	switch enc {
	case "rgb8":
		s := inferStep(3, 1)
		dst := image.NewNRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			src := data[y*s : y*s+width*3]
			di, si := y*dst.Stride, 0
			for x := 0; x < width; x++ {
				dst.Pix[di+0] = src[si+0] // R
				dst.Pix[di+1] = src[si+1] // G
				dst.Pix[di+2] = src[si+2] // B
				dst.Pix[di+3] = 0xFF
				di += 4
				si += 3
			}
		}
		img = dst

	case "bgr8":
		s := inferStep(3, 1)
		dst := image.NewNRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			src := data[y*s : y*s+width*3]
			di, si := y*dst.Stride, 0
			for x := 0; x < width; x++ {
				dst.Pix[di+0] = src[si+2] // R
				dst.Pix[di+1] = src[si+1] // G
				dst.Pix[di+2] = src[si+0] // B
				dst.Pix[di+3] = 0xFF
				di += 4
				si += 3
			}
		}
		img = dst

	case "rgba8":
		s := inferStep(4, 1)
		dst := image.NewNRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			copy(dst.Pix[y*dst.Stride:y*dst.Stride+width*4], data[y*s:y*s+width*4])
		}
		img = dst

	case "bgra8":
		s := inferStep(4, 1)
		dst := image.NewNRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			src := data[y*s : y*s+width*4]
			di, si := y*dst.Stride, 0
			for x := 0; x < width; x++ {
				b, g, r, a := src[si+0], src[si+1], src[si+2], src[si+3]
				dst.Pix[di+0] = r
				dst.Pix[di+1] = g
				dst.Pix[di+2] = b
				dst.Pix[di+3] = a
				di += 4
				si += 4
			}
		}
		img = dst

	case "mono8", "8uc1":
		s := inferStep(1, 1)
		dst := image.NewGray(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			copy(dst.Pix[y*dst.Stride:y*dst.Stride+width], data[y*s:y*s+width])
		}
		img = dst

	case "mono16", "16uc1":
		// For PNG we can preserve 16-bit; for JPEG we must map to 8-bit.
		s := inferStep(1, 2)
		if fmtFmt == "png" {
			dst := image.NewGray16(image.Rect(0, 0, width, height))
			for y := 0; y < height; y++ {
				for x := 0; x < width; x++ {
					o := y*s + 2*x
					lo, hi := data[o], data[o+1]
					var v uint16
					if isBigEndian {
						v = (uint16(hi) << 8) | uint16(lo)
					} else {
						v = (uint16(lo) << 8) | uint16(hi)
					}
					p := y*dst.Stride + 2*x
					dst.Pix[p+0] = byte(v >> 8)
					dst.Pix[p+1] = byte(v)
				}
			}
			img = dst
		} else { // jpeg: scale to 8-bit
			// find min/max (ignore 0)
			minv, maxv := uint16(0xFFFF), uint16(0)
			for y := 0; y < height; y++ {
				for x := 0; x < width; x++ {
					o := y*s + 2*x
					lo, hi := data[o], data[o+1]
					var v uint16
					if isBigEndian {
						v = (uint16(hi) << 8) | uint16(lo)
					} else {
						v = (uint16(lo) << 8) | uint16(hi)
					}
					if v != 0 {
						if v < minv {
							minv = v
						}
						if v > maxv {
							maxv = v
						}
					}
				}
			}
			dst := image.NewGray(image.Rect(0, 0, width, height))
			if maxv > minv {
				scale := 255.0 / float64(maxv-minv)
				for y := 0; y < height; y++ {
					for x := 0; x < width; x++ {
						o := y*s + 2*x
						lo, hi := data[o], data[o+1]
						var v uint16
						if isBigEndian {
							v = (uint16(hi) << 8) | uint16(lo)
						} else {
							v = (uint16(lo) << 8) | uint16(hi)
						}
						var g byte
						if v > 0 {
							g = byte(float64(v-minv) * scale)
						}
						dst.Pix[y*dst.Stride+x] = g
					}
				}
			}
			img = dst
		}

	case "32fc1":
		// Scale float depth to Gray16 (PNG) or Gray8 (JPEG)
		s := inferStep(1, 4)
		readF32 := func(o int) float32 {
			if isBigEndian {
				bits := uint32(data[o])<<24 |
					uint32(data[o+1])<<16 |
					uint32(data[o+2])<<8 |
					uint32(data[o+3])
				return math.Float32frombits(bits)
			}
			bits := uint32(data[o]) |
				uint32(data[o+1])<<8 |
				uint32(data[o+2])<<16 |
				uint32(data[o+3])<<24
			return math.Float32frombits(bits)
		}
		minf, maxf := math.Inf(1), math.Inf(-1)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				v := float64(readF32(y*s + 4*x))
				if v > 0 && !math.IsNaN(v) && !math.IsInf(v, 0) {
					if v < minf {
						minf = v
					}
					if v > maxf {
						maxf = v
					}
				}
			}
		}
		if fmtFmt == "png" {
			dst := image.NewGray16(image.Rect(0, 0, width, height))
			if maxf > minf && !math.IsInf(minf, 0) && !math.IsInf(maxf, 0) {
				scale := 65535.0 / (maxf - minf)
				for y := 0; y < height; y++ {
					for x := 0; x < width; x++ {
						v := float64(readF32(y*s + 4*x))
						var s16 uint16
						if v > 0 && !math.IsNaN(v) && !math.IsInf(v, 0) {
							s16 = uint16((v - minf) * scale)
						}
						p := y*dst.Stride + 2*x
						dst.Pix[p+0] = byte(s16 >> 8)
						dst.Pix[p+1] = byte(s16)
					}
				}
			}
			img = dst
		} else { // jpeg
			dst := image.NewGray(image.Rect(0, 0, width, height))
			if maxf > minf && !math.IsInf(minf, 0) && !math.IsInf(maxf, 0) {
				scale := 255.0 / (maxf - minf)
				for y := 0; y < height; y++ {
					for x := 0; x < width; x++ {
						v := float64(readF32(y*s + 4*x))
						var g byte
						if v > 0 && !math.IsNaN(v) && !math.IsInf(v, 0) {
							g = byte((v - minf) * scale)
						}
						dst.Pix[y*dst.Stride+x] = g
					}
				}
			}
			img = dst
		}

	default:
		if strings.HasPrefix(enc, "bayer") || strings.HasPrefix(enc, "rgb") || strings.HasPrefix(enc, "bgr") {
			s := inferStep(1, 1)
			dst := image.NewGray(image.Rect(0, 0, width, height))
			for y := 0; y < height; y++ {
				src := data[y*s:]
				for x := 0; x < width; x++ {
					dst.SetGray(x, y, color.Gray{Y: src[x]})
				}
			}
			img = dst
		} else {
			return nil, fmt.Errorf("unsupported encoding %q", encoding)
		}
	}

	var buf bytes.Buffer
	if fmtFmt == "png" {
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	} else {
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}
