package functions

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/image/draw"
)

// Rename cleans up the filenames in the given folder. It removes special
// characters and prefixes the filename with a timestamp.
func Rename(timestamp int64, baseDir, imageType string) error {
	files, err := os.ReadDir(baseDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()

		filename = strings.ToLower(filename)
		filename = strings.ReplaceAll(filename, "-", "_")
		filename = strings.ReplaceAll(filename, " ", "_")
		filename = strings.ReplaceAll(filename, "ö", "oe")
		filename = strings.ReplaceAll(filename, "ü", "ue")
		filename = strings.ReplaceAll(filename, "ä", "ae")

		data, err := os.ReadFile(filepath.Join(baseDir, file.Name()))
		if err != nil {
			return err
		}

		if err := os.WriteFile(filepath.Join(baseDir, fmt.Sprintf("%d_%s_%s", timestamp, imageType, filename)), data, 0644); err != nil {
			return err
		}
	}

	return nil
}

// Convert uses Inkscape to convert svg files into png files. It only takes
// into account files that have the given timestamp prefix.
func Convert(timestamp int64, baseDir string, targetWidth, targetHeight int) error {
	files, err := os.ReadDir(baseDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), fmt.Sprintf("%d", timestamp)) && filepath.Ext(file.Name()) == ".svg" {
			if _, err := convertToPng(filepath.Join(baseDir, file.Name()), targetWidth, targetHeight); err != nil {
				return err
			}
		}
	}

	return nil
}

// Resize resizes all jpeg/png files from the folder that have timestamp as
// prefix.
func Resize(timestamp int64, baseDir string, targetWidth, targetHeight int) error {
	files, err := os.ReadDir(baseDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), fmt.Sprintf("%d", timestamp)) {
			switch filepath.Ext(file.Name()) {
			case ".png":
				fallthrough
			case ".jpeg":
				fallthrough
			case ".jpg":
				if err := resizeImage(filepath.Join(baseDir, file.Name()), targetWidth, targetHeight); err != nil {
					return err
				}

			default:
				fmt.Printf("image format from '%s' is not supported for resizing\n", file.Name())
			}
		}
	}

	return nil
}

// CopyFiles copies files from one folder to another. It only copies files if
// they have timestamp as a prefix.
func CopyFiles(timestamp int64, baseDir, targetDir string) error {
	files, err := os.ReadDir(baseDir)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), fmt.Sprintf("%d", timestamp)) {
			data, err := os.ReadFile(filepath.Join(baseDir, file.Name()))
			if err != nil {
				return err
			}

			name := file.Name()
			name = strings.ReplaceAll(name, "_resized", "")

			if err := os.WriteFile(filepath.Join(targetDir, name), data, 0644); err != nil {
				return err
			}

			if err := os.Remove(filepath.Join(baseDir, file.Name())); err != nil {
				return err
			}
		}
	}

	return nil
}

func getDimensions(filename string) (float64, float64, error) {
	var width, height float64

	res, err := exec.Command("inkscape", "-W", filename).CombinedOutput()
	if err != nil {
		return width, height, errors.Wrap(err, "read svg width")
	}

	width, err = strconv.ParseFloat(strings.TrimSpace(string(res)), 64)
	if err != nil {
		return width, height, errors.Wrap(err, "parse svg width")
	}

	res, err = exec.Command("inkscape", "-H", filename).CombinedOutput()
	if err != nil {
		return width, height, errors.Wrap(err, "read svg height")
	}

	height, err = strconv.ParseFloat(strings.TrimSpace(string(res)), 64)
	if err != nil {
		return width, height, errors.Wrap(err, "parse svg height")
	}

	return width, height, nil
}

func convertToPng(filename string, targetWidth, targetHeight int) (string, error) {
	var width, height string

	w, h, err := getDimensions(filename)
	if err != nil {
		return "", errors.Wrap(err, "check svg dimensions")
	}

	factorX := int(w) / targetWidth
	factorY := int(h) / targetHeight

	if factorX > factorY {
		width = fmt.Sprintf("%d", int(w)/factorX)
		height = fmt.Sprintf("%d", int(h)/factorX)
	} else {
		width = fmt.Sprintf("%d", int(w)/factorY)
		height = fmt.Sprintf("%d", int(h)/factorY)
	}

	target := strings.ReplaceAll(filename, ".svg", ".png")

	if err := exec.Command("inkscape", "-w", width, "-h", height, filename, "-o", target).Run(); err != nil {
		return "", errors.Wrap(err, "convert svg to png")
	}

	return target, nil
}

func resizeImage(filename string, targetWidth, targetHeight int) error {
	var (
		output bytes.Buffer
		src    image.Image
	)

	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(data)

	switch filepath.Ext(filename) {
	case ".png":
		src, err = png.Decode(buf)

	case ".jpg":
		fallthrough
	case ".jpeg":
		src, err = jpeg.Decode(buf)

	default:
		log.Println("filetype currently not supported, skipping " + filename)
		return nil
	}

	if err != nil {
		return err
	}

	x := float64(src.Bounds().Max.X)
	y := float64(src.Bounds().Max.Y)

	factorX := x / float64(targetWidth)
	factorY := y / float64(targetHeight)

	var dst *image.RGBA

	if factorX > factorY {
		dst = image.NewRGBA(image.Rect(0, 0, targetWidth, int(y/factorX)))
	} else {
		dst = image.NewRGBA(image.Rect(0, 0, int(x/factorY), targetHeight))
	}

	draw.NearestNeighbor.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)
	if err := png.Encode(&output, dst); err != nil {
		return err
	}

	return os.WriteFile(strings.ReplaceAll(filename, filepath.Ext(filename), fmt.Sprintf("_resized%s", filepath.Ext(filename))), output.Bytes(), 0644)
}
