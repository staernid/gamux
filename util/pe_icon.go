package util

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExtractPEIcon extracts the highest-resolution icon embedded in a Windows PE executable (.exe)
// and writes it to destPath as a PNG (or ICO if conversion is not possible).
func ExtractPEIcon(exePath, destPath string) error {
	if exePath == "" || destPath == "" {
		return fmt.Errorf("exePath and destPath cannot be empty")
	}

	data, format, err := ExtractPEIconBytes(exePath)
	if err != nil {
		// Fallback to system tools if installed (wrestool or icoextract)
		if fallbackErr := extractIconWithExternalTools(exePath, destPath); fallbackErr == nil {
			return nil
		}
		return fmt.Errorf("extract icon from %s: %w", exePath, err)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}

	// If destination expects PNG but format is ICO, attempt conversion or write directly
	if strings.HasSuffix(strings.ToLower(destPath), ".png") && format == "ico" {
		if pngBytes, err := convertBmpIconToPNG(data); err == nil && len(pngBytes) > 0 {
			data = pngBytes
		}
	}

	return os.WriteFile(destPath, data, 0644)
}

// ExtractPEIconBytes extracts the best icon embedded in a Windows PE binary (.exe)
// and returns the raw image bytes and format ("png" or "ico").
func ExtractPEIconBytes(exePath string) ([]byte, string, error) {
	f, err := os.Open(exePath)
	if err != nil {
		return nil, "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	return ExtractPEIconFromReader(f)
}

// ExtractPEIconFromReader extracts the best icon from an io.ReaderAt, ensuring testability.
func ExtractPEIconFromReader(r io.ReaderAt) ([]byte, string, error) {
	peFile, err := pe.NewFile(r)
	if err != nil {
		return nil, "", fmt.Errorf("parse PE file: %w", err)
	}
	defer peFile.Close()

	var rsrcSec *pe.Section
	for _, sec := range peFile.Sections {
		if strings.TrimRight(sec.Name, "\x00") == ".rsrc" {
			rsrcSec = sec
			break
		}
	}
	if rsrcSec == nil {
		return nil, "", fmt.Errorf("no .rsrc section found in executable")
	}

	rsrcData, err := rsrcSec.Data()
	if err != nil {
		return nil, "", fmt.Errorf("read .rsrc section data: %w", err)
	}

	return extractBestIconFromRsrc(rsrcData, rsrcSec.VirtualAddress)
}

type rsrcEntry struct {
	id     uint32
	offset uint32
	isDir  bool
}

func readRsrcDirectory(data []byte, offset uint32) ([]rsrcEntry, error) {
	if int(offset)+16 > len(data) {
		return nil, fmt.Errorf("directory offset out of bounds")
	}
	namedEntries := binary.LittleEndian.Uint16(data[offset+12 : offset+14])
	idEntries := binary.LittleEndian.Uint16(data[offset+14 : offset+16])
	total := int(namedEntries) + int(idEntries)

	var entries []rsrcEntry
	pos := int(offset) + 16
	for i := 0; i < total && pos+8 <= len(data); i++ {
		entryID := binary.LittleEndian.Uint32(data[pos : pos+4])
		entryOff := binary.LittleEndian.Uint32(data[pos+4 : pos+8])
		pos += 8
		entries = append(entries, rsrcEntry{
			id:     entryID,
			offset: entryOff & 0x7FFFFFFF,
			isDir:  (entryOff & 0x80000000) != 0,
		})
	}
	return entries, nil
}

func readRsrcDataEntry(data []byte, offset uint32) (rva uint32, size uint32, err error) {
	if int(offset)+16 > len(data) {
		return 0, 0, fmt.Errorf("data entry offset out of bounds")
	}
	rva = binary.LittleEndian.Uint32(data[offset : offset+4])
	size = binary.LittleEndian.Uint32(data[offset+4 : offset+8])
	return rva, size, nil
}

type grpIconDirEntry struct {
	width      byte
	height     byte
	colorCount byte
	reserved   byte
	planes     uint16
	bitCount   uint16
	bytesInRes uint32
	id         uint16
}

func extractBestIconFromRsrc(rsrcData []byte, rsrcVirtualAddress uint32) ([]byte, string, error) {
	rootEntries, err := readRsrcDirectory(rsrcData, 0)
	if err != nil {
		return nil, "", fmt.Errorf("read root directory: %w", err)
	}

	var groupIconDirOff, iconDirOff uint32
	var hasGroupIcon, hasIcon bool
	for _, e := range rootEntries {
		if e.id == 14 { // RT_GROUP_ICON
			groupIconDirOff = e.offset
			hasGroupIcon = true
		} else if e.id == 3 { // RT_ICON
			iconDirOff = e.offset
			hasIcon = true
		}
	}

	if !hasGroupIcon || !hasIcon {
		return nil, "", fmt.Errorf("executable does not contain icon resources")
	}

	// Traverse RT_GROUP_ICON to locate the first group's data entry
	groupEntries, err := readRsrcDirectory(rsrcData, groupIconDirOff)
	if err != nil || len(groupEntries) == 0 {
		return nil, "", fmt.Errorf("read group icon directory: %w", err)
	}

	firstGroup := groupEntries[0]
	groupDataOff := firstGroup.offset
	if firstGroup.isDir {
		langEntries, err := readRsrcDirectory(rsrcData, firstGroup.offset)
		if err != nil || len(langEntries) == 0 {
			return nil, "", fmt.Errorf("read group icon lang directory: %w", err)
		}
		groupDataOff = langEntries[0].offset
	}

	groupRVA, groupSize, err := readRsrcDataEntry(rsrcData, groupDataOff)
	if err != nil {
		return nil, "", fmt.Errorf("read group data entry: %w", err)
	}

	if groupRVA < rsrcVirtualAddress {
		return nil, "", fmt.Errorf("invalid group icon RVA")
	}
	groupFileOffset := groupRVA - rsrcVirtualAddress
	if int(groupFileOffset+groupSize) > len(rsrcData) || groupSize < 6 {
		return nil, "", fmt.Errorf("group icon data exceeds resource section bounds")
	}

	groupData := rsrcData[groupFileOffset : groupFileOffset+groupSize]
	idCount := binary.LittleEndian.Uint16(groupData[4:6])
	if idCount == 0 {
		return nil, "", fmt.Errorf("group icon contains 0 frames")
	}

	// Parse frame descriptors and choose the best quality
	var best grpIconDirEntry
	var bestScore int
	for i := 0; i < int(idCount); i++ {
		off := 6 + i*14
		if off+14 > len(groupData) {
			break
		}
		e := grpIconDirEntry{
			width:      groupData[off],
			height:     groupData[off+1],
			colorCount: groupData[off+2],
			reserved:   groupData[off+3],
			planes:     binary.LittleEndian.Uint16(groupData[off+4 : off+6]),
			bitCount:   binary.LittleEndian.Uint16(groupData[off+6 : off+8]),
			bytesInRes: binary.LittleEndian.Uint32(groupData[off+8 : off+12]),
			id:         binary.LittleEndian.Uint16(groupData[off+12 : off+14]),
		}

		w := int(e.width)
		if w == 0 {
			w = 256
		}
		h := int(e.height)
		if h == 0 {
			h = 256
		}
		score := w * h * int(e.bitCount)
		if score > bestScore {
			bestScore = score
			best = e
		}
	}

	if best.id == 0 && bestScore == 0 {
		return nil, "", fmt.Errorf("no valid icon frames in group")
	}

	// Locate matching RT_ICON entry
	iconEntries, err := readRsrcDirectory(rsrcData, iconDirOff)
	if err != nil {
		return nil, "", fmt.Errorf("read icon directory: %w", err)
	}

	var targetIconEntryOff uint32
	var foundIcon bool
	for _, ie := range iconEntries {
		if ie.id == uint32(best.id) {
			targetIconEntryOff = ie.offset
			foundIcon = true
			break
		}
	}
	if !foundIcon {
		return nil, "", fmt.Errorf("icon ID %d not found in RT_ICON directory", best.id)
	}

	iconDataOff := targetIconEntryOff
	langEntries, err := readRsrcDirectory(rsrcData, targetIconEntryOff)
	if err == nil && len(langEntries) > 0 {
		iconDataOff = langEntries[0].offset
	}

	iconRVA, iconSize, err := readRsrcDataEntry(rsrcData, iconDataOff)
	if err != nil {
		return nil, "", fmt.Errorf("read icon data entry: %w", err)
	}

	if iconRVA < rsrcVirtualAddress {
		return nil, "", fmt.Errorf("invalid icon RVA")
	}
	iconSecOffset := iconRVA - rsrcVirtualAddress
	if int(iconSecOffset+iconSize) > len(rsrcData) {
		return nil, "", fmt.Errorf("icon data exceeds resource section bounds")
	}

	rawPayload := rsrcData[iconSecOffset : iconSecOffset+iconSize]

	// Modern Windows executables often embed raw PNG data directly
	if bytes.HasPrefix(rawPayload, []byte("\x89PNG\r\n\x1a\n")) {
		return rawPayload, "png", nil
	}

	// Try converting 32-bit / 24-bit BMP to PNG for optimal rendering
	w := int(best.width)
	if w == 0 {
		w = 256
	}
	h := int(best.height)
	if h == 0 {
		h = 256
	}

	if pngBytes, err := convertRawBmpPayloadToPNG(rawPayload, w, h); err == nil && len(pngBytes) > 0 {
		return pngBytes, "png", nil
	}

	// Construct standard .ico file containing the frame
	var icoBuf bytes.Buffer
	// ICONDIR: Reserved(2), Type(2), Count(2)
	icoBuf.Write([]byte{0, 0, 1, 0, 1, 0})
	// ICONDIRENTRY: 16 bytes
	icoBuf.WriteByte(best.width)
	icoBuf.WriteByte(best.height)
	icoBuf.WriteByte(best.colorCount)
	icoBuf.WriteByte(best.reserved)
	icoBuf.Write([]byte{byte(best.planes), byte(best.planes >> 8)})
	icoBuf.Write([]byte{byte(best.bitCount), byte(best.bitCount >> 8)})
	var szBuf [4]byte
	binary.LittleEndian.PutUint32(szBuf[:], uint32(len(rawPayload)))
	icoBuf.Write(szBuf[:])
	var offBuf [4]byte
	binary.LittleEndian.PutUint32(offBuf[:], 22) // header (6) + entry (16)
	icoBuf.Write(offBuf[:])
	icoBuf.Write(rawPayload)

	return icoBuf.Bytes(), "ico", nil
}

// convertRawBmpPayloadToPNG parses 32-bit or 24-bit BITMAPINFOHEADER and converts to PNG.
func convertRawBmpPayloadToPNG(raw []byte, width, height int) ([]byte, error) {
	if len(raw) < 40 {
		return nil, fmt.Errorf("BMP payload too small")
	}

	bpp := binary.LittleEndian.Uint16(raw[14:16])
	if bpp != 32 && bpp != 24 {
		return nil, fmt.Errorf("unsupported bpp for direct PNG conversion: %d", bpp)
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	pixelData := raw[40:]
	bytesPerPixel := int(bpp) / 8
	rowBytes := ((width*bytesPerPixel + 3) / 4) * 4

	for y := 0; y < height; y++ {
		srcRow := height - 1 - y
		srcOffset := srcRow * rowBytes
		if srcOffset+rowBytes > len(pixelData) {
			break
		}
		for x := 0; x < width; x++ {
			pxOff := srcOffset + x*bytesPerPixel
			b := pixelData[pxOff]
			g := pixelData[pxOff+1]
			r := pixelData[pxOff+2]
			a := byte(255)
			if bytesPerPixel == 4 {
				a = pixelData[pxOff+3]
			}
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func convertBmpIconToPNG(icoData []byte) ([]byte, error) {
	if len(icoData) < 22 {
		return nil, fmt.Errorf("ICO data too short")
	}
	w := int(icoData[6])
	if w == 0 {
		w = 256
	}
	h := int(icoData[7])
	if h == 0 {
		h = 256
	}
	rawPayload := icoData[22:]
	return convertRawBmpPayloadToPNG(rawPayload, w, h)
}

func extractIconWithExternalTools(exePath, destPath string) error {
	// 1. Try icoextract if available
	if icoPath, err := exec.LookPath("icoextract"); err == nil && icoPath != "" {
		cmd := exec.Command(icoPath, exePath, destPath)
		if err := cmd.Run(); err == nil && FileExists(destPath) {
			slog.Debug("Extracted icon using icoextract", "exe", exePath, "dest", destPath)
			return nil
		}
	}

	// 2. Try wrestool (from icoutils) if available
	if wrestoolPath, err := exec.LookPath("wrestool"); err == nil && wrestoolPath != "" {
		cmd := exec.Command(wrestoolPath, "-x", "-t14", exePath, "-o", destPath)
		if err := cmd.Run(); err == nil && FileExists(destPath) {
			slog.Debug("Extracted icon using wrestool", "exe", exePath, "dest", destPath)
			return nil
		}
	}

	return fmt.Errorf("no external icon extraction tools available")
}
