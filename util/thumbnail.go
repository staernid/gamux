package util

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// thumbSem limits concurrent image decoding and resizing to avoid memory and CPU saturation.
var thumbSem = make(chan struct{}, 4)

// thumbMu synchronizes disk cache generation for the same target file.
var (
	thumbMu  sync.Mutex
	inFlight = make(map[string]chan struct{})
)

// ResizeImage resizes an image to the specified targetWidth using area-averaging (box filter),
// preserving aspect ratio. If targetWidth >= original width, the original image is copied to RGBA.
func ResizeImage(src image.Image, targetWidth int) *image.RGBA {
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW <= 0 || srcH <= 0 || targetWidth <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}

	if targetWidth >= srcW {
		dst := image.NewRGBA(image.Rect(0, 0, srcW, srcH))
		for y := 0; y < srcH; y++ {
			for x := 0; x < srcW; x++ {
				dst.Set(x, y, src.At(bounds.Min.X+x, bounds.Min.Y+y))
			}
		}
		return dst
	}

	targetHeight := int(math.Round(float64(targetWidth) * float64(srcH) / float64(srcW)))
	if targetHeight < 1 {
		targetHeight = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))

	xScale := float64(srcW) / float64(targetWidth)
	yScale := float64(srcH) / float64(targetHeight)

	for dy := 0; dy < targetHeight; dy++ {
		syStart := int(float64(dy) * yScale)
		syEnd := int(float64(dy+1) * yScale)
		if syEnd <= syStart {
			syEnd = syStart + 1
		}
		if syEnd > srcH {
			syEnd = srcH
		}

		for dx := 0; dx < targetWidth; dx++ {
			sxStart := int(float64(dx) * xScale)
			sxEnd := int(float64(dx+1) * xScale)
			if sxEnd <= sxStart {
				sxEnd = sxStart + 1
			}
			if sxEnd > srcW {
				sxEnd = srcW
			}

			var rSum, gSum, bSum, aSum uint64
			var count uint64

			for sy := syStart; sy < syEnd; sy++ {
				for sx := sxStart; sx < sxEnd; sx++ {
					r, g, b, a := src.At(bounds.Min.X+sx, bounds.Min.Y+sy).RGBA()
					rSum += uint64(r)
					gSum += uint64(g)
					bSum += uint64(b)
					aSum += uint64(a)
					count++
				}
			}

			if count > 0 {
				dst.SetRGBA(dx, dy, color.RGBA{
					R: uint8((rSum / count) >> 8),
					G: uint8((gSum / count) >> 8),
					B: uint8((bSum / count) >> 8),
					A: uint8((aSum / count) >> 8),
				})
			}
		}
	}

	return dst
}

// GenerateThumbnail decodes srcPath, resizes to targetWidth, and writes dstPath atomically.
func GenerateThumbnail(srcPath, dstPath string, targetWidth int) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source image: %w", err)
	}
	defer f.Close()

	srcImg, format, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	dstImg := ResizeImage(srcImg, targetWidth)

	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("create thumbnail dir: %w", err)
	}

	tmpFile := fmt.Sprintf("%s.%d.tmp", dstPath, os.Getpid())
	out, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("create temp thumbnail: %w", err)
	}

	writeErr := func() error {
		defer out.Close()
		if strings.EqualFold(format, "png") {
			return png.Encode(out, dstImg)
		}
		return jpeg.Encode(out, dstImg, &jpeg.Options{Quality: 85})
	}()

	if writeErr != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("encode thumbnail: %w", writeErr)
	}

	if err := os.Rename(tmpFile, dstPath); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("commit thumbnail: %w", err)
	}

	return nil
}

// GetOrGenerateThumbnail returns the path to a cached thumbnail for srcPath resized to targetWidth.
// If the cached thumbnail does not exist or is older than srcPath, it is generated under cacheDir.
func GetOrGenerateThumbnail(srcPath string, targetWidth int, cacheDir string) (string, error) {
	if srcPath == "" {
		return "", fmt.Errorf("empty source path")
	}
	if targetWidth <= 0 {
		targetWidth = 300
	}
	if targetWidth < 32 {
		targetWidth = 32
	}
	if targetWidth > 1200 {
		targetWidth = 1200
	}

	fi, err := os.Stat(srcPath)
	if err != nil {
		return "", fmt.Errorf("stat source image: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(srcPath))
	if ext != ".png" {
		ext = ".jpg"
	}

	h := sha256.Sum256([]byte(srcPath))
	hashKey := fmt.Sprintf("%x", h[:12])
	thumbFilename := fmt.Sprintf("%s_w%d_m%d%s", hashKey, targetWidth, fi.ModTime().Unix(), ext)
	thumbPath := filepath.Join(cacheDir, thumbFilename)

	if FileExists(thumbPath) {
		return thumbPath, nil
	}

	// Single-flight deduplication per thumbnail path to prevent duplicate simultaneous encodes
	thumbMu.Lock()
	waitCh, exists := inFlight[thumbPath]
	if exists {
		thumbMu.Unlock()
		<-waitCh
		if FileExists(thumbPath) {
			return thumbPath, nil
		}
		thumbMu.Lock()
	}
	doneCh := make(chan struct{})
	inFlight[thumbPath] = doneCh
	thumbMu.Unlock()

	defer func() {
		thumbMu.Lock()
		delete(inFlight, thumbPath)
		close(doneCh)
		thumbMu.Unlock()
	}()

	// Acquire concurrency slot
	thumbSem <- struct{}{}
	defer func() { <-thumbSem }()

	if err := GenerateThumbnail(srcPath, thumbPath, targetWidth); err != nil {
		return "", err
	}

	return thumbPath, nil
}
