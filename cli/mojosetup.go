package cli

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/boggydigital/pathways"
)

const (
	fsPfx     = "filesizes="
	scriptSfx = "eval $finish; exit $res"
	scriptFn  = "script.sh"
	redistFn  = "mojosetup.tar.gz"
	DataFn    = "data.zip"
)

type ExtractOptions struct {
	Src    string
	Dst    string
	Script bool
	Redist bool
	Data   bool
}

func MojosetupExtract(opt *ExtractOptions) error {

	if _, err := os.Stat(opt.Dst); os.IsNotExist(err) {
		if err = os.MkdirAll(opt.Dst, pathways.PermUrwGrwOr); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	fInput, err := os.Open(opt.Src)
	if err != nil {
		return err
	}
	defer fInput.Close()

	var scriptLen int64
	var fsLen int64

	scanner := bufio.NewScanner(fInput)

	for scanner.Scan() {
		scriptLen += int64(len(scanner.Bytes()) + len("\n"))

		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, fsPfx) {
			if _, fsStr, ok := strings.Cut(line, "="); ok {
				fsStr = strings.Trim(fsStr, "\"")
				if fsLen, err = strconv.ParseInt(fsStr, 10, 64); err != nil {
					return err
				}
			}
		} else if line == scriptSfx {
			break
		}
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	if scriptLen == 0 {
		return errors.New("mojosetup installer script not present")
	}

	if fsLen == 0 {
		return errors.New("mojosetup redistributable not present")
	}

	var stat os.FileInfo
	if stat, err = fInput.Stat(); err == nil {
		if stat.Size() <= fsLen {
			return errors.New("mojosetup data not present")
		}
	}

	if _, err = fInput.Seek(0, io.SeekStart); err != nil {
		return err
	}

	if err = copyOrSkipLen(fInput, scriptLen, opt.Script, opt.Dst, scriptFn); err != nil {
		return err
	}

	if err = copyOrSkipLen(fInput, fsLen, opt.Redist, opt.Dst, redistFn); err != nil {
		return err
	}

	if err = copyOrSkipLen(fInput, 0, opt.Data, opt.Dst, DataFn); err != nil {
		return err
	}

	return nil
}

func copyOrSkipLen(rs io.ReadSeeker, limit int64, flag bool, pathParts ...string) error {

	if limit < 0 {
		return errors.New("copyOrSkipLen limit must be zero or positive")
	}

	var err error

	switch flag {
	case true:

		if len(pathParts) == 0 {
			return errors.New("copyOrSkipLen requires valid output path")
		}

		var extractedFile io.WriteCloser
		extractedFile, err = os.Create(filepath.Join(pathParts...))
		if err != nil {
			return err
		}
		defer extractedFile.Close()

		switch limit {
		case 0:
			_, err = io.Copy(extractedFile, rs)
		default:
			_, err = io.CopyN(extractedFile, rs, limit)
		}
	default:
		_, err = rs.Seek(limit, io.SeekCurrent)
	}

	return err
}
