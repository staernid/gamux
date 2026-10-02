package util

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func createTestImage(t *testing.T, path string, width, height int, isPNG bool) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: 128,
				A: 255,
			})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if isPNG {
		if err := png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
	} else {
		if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResizeImage(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1000, 500))
	for y := 0; y < 500; y++ {
		for x := 0; x < 1000; x++ {
			src.SetRGBA(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}

	// 1. Downscale to 200px width
	dst := ResizeImage(src, 200)
	if dst.Bounds().Dx() != 200 {
		t.Errorf("expected width 200, got %d", dst.Bounds().Dx())
	}
	if dst.Bounds().Dy() != 100 {
		t.Errorf("expected height 100, got %d", dst.Bounds().Dy())
	}

	c := dst.RGBAAt(50, 50)
	if c.R < 190 || c.R > 210 || c.G < 90 || c.G > 110 {
		t.Errorf("unexpected color after downscale: %+v", c)
	}

	// 2. Target width >= source width returns same dimensions without upscaling
	dstSame := ResizeImage(src, 1200)
	if dstSame.Bounds().Dx() != 1000 || dstSame.Bounds().Dy() != 500 {
		t.Errorf("expected original dimensions 1000x500, got %dx%d", dstSame.Bounds().Dx(), dstSame.Bounds().Dy())
	}

	// 3. Zero / invalid dimensions handling
	empty := ResizeImage(src, 0)
	if empty.Bounds().Dx() != 1 || empty.Bounds().Dy() != 1 {
		t.Errorf("expected 1x1 fallback for zero width")
	}
}

func TestGenerateThumbnail(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.jpg")
	dstPath := filepath.Join(tmpDir, "thumb.jpg")

	createTestImage(t, srcPath, 800, 600, false)

	if err := GenerateThumbnail(srcPath, dstPath, 200); err != nil {
		t.Fatalf("GenerateThumbnail failed: %v", err)
	}

	if !FileExists(dstPath) {
		t.Fatalf("expected thumbnail to exist at %s", dstPath)
	}

	// Verify thumbnail can be decoded and has expected dimensions
	f, err := os.Open(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	cfg, format, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatalf("failed to decode thumbnail config: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("expected format jpeg, got %s", format)
	}
	if cfg.Width != 200 {
		t.Errorf("expected thumb width 200, got %d", cfg.Width)
	}
	if cfg.Height != 150 {
		t.Errorf("expected thumb height 150, got %d", cfg.Height)
	}
}

func TestGetOrGenerateThumbnail_CachingAndInvalidation(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.png")
	cacheDir := filepath.Join(tmpDir, "cache")

	createTestImage(t, srcPath, 600, 900, true)

	// 1. Initial generation
	thumb1, err := GetOrGenerateThumbnail(srcPath, 300, cacheDir)
	if err != nil {
		t.Fatalf("first GetOrGenerateThumbnail failed: %v", err)
	}
	if !FileExists(thumb1) {
		t.Fatalf("thumb1 does not exist: %s", thumb1)
	}

	info1, err := os.Stat(thumb1)
	if err != nil {
		t.Fatal(err)
	}

	// 2. Immediate second request should hit cache
	thumb2, err := GetOrGenerateThumbnail(srcPath, 300, cacheDir)
	if err != nil {
		t.Fatalf("second GetOrGenerateThumbnail failed: %v", err)
	}
	if thumb1 != thumb2 {
		t.Errorf("expected same cache path, got %s and %s", thumb1, thumb2)
	}
	info2, _ := os.Stat(thumb2)
	if !info1.ModTime().Equal(info2.ModTime()) {
		t.Errorf("cache file was modified on cache hit")
	}

	// 3. Modifying source image modtime invalidates cache and yields new thumbnail
	time.Sleep(10 * time.Millisecond)
	newTime := time.Now().Add(5 * time.Second)
	_ = os.Chtimes(srcPath, newTime, newTime)

	thumb3, err := GetOrGenerateThumbnail(srcPath, 300, cacheDir)
	if err != nil {
		t.Fatalf("third GetOrGenerateThumbnail failed: %v", err)
	}
	if thumb3 == thumb1 {
		t.Errorf("expected new thumbnail filename after source mtime changed, got %s", thumb3)
	}
}

func TestGetOrGenerateThumbnail_Concurrent(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "concurrent.jpg")
	cacheDir := filepath.Join(tmpDir, "cache")

	createTestImage(t, srcPath, 1200, 800, false)

	var wg sync.WaitGroup
	results := make([]string, 10)
	errs := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			path, err := GetOrGenerateThumbnail(srcPath, 300, cacheDir)
			results[idx] = path
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	for i := 0; i < 10; i++ {
		if errs[i] != nil {
			t.Errorf("worker %d returned error: %v", i, errs[i])
		}
		if results[i] != results[0] {
			t.Errorf("worker %d returned divergent path %s, want %s", i, results[i], results[0])
		}
	}
}
