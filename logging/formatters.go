package logging

import (
	"github.com/fatih/color"
	"os"
	"path/filepath"
	"strings"
)

func formatFieldName(i any) string {
	return color.CyanString("%s=", i)
}
func formatFieldValue(i any) string {
	return color.YellowString("%s", i)
}
func formatErrFieldValue(i any) string {
	return color.RedString("%s=", i)
}

func defaultFormatCaller(i any) string {
	projectRoot, err := os.Getwd()
	if err != nil {
		projectRoot = ""
	}
	fullPath, ok := i.(string)
	if !ok {
		return ""
	}
	relPath := fullPath
	if strings.Contains(fullPath, "@") {
		relPath = func() string {
			parts := strings.Split(filepath.ToSlash(fullPath), "/")
			for i, part := range parts {
				if strings.Contains(part, "@") {
					slice := parts[i:]
					slice[0] = "@" + strings.Split(part, "@")[0]
					return strings.Join(slice, string(filepath.Separator))
				}
			}
			// If not found, return the original path
			return fullPath
		}()
	} else if rel, err := filepath.Rel(projectRoot, fullPath); err == nil {
		relPath = rel
	}

	return color.HiBlackString("%s", relPath)
}
func ergoFormatCaller(_ any) string {
	return color.HiBlackString("ErgoAM")
}
func fxFormatCaller(_ any) string {
	return color.HiBlackString("UberFX")
}
