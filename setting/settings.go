// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package setting

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/OpenPeeDeeP/xdg"
	"github.com/nelsam/gxui"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
)

const (
	LicenseHeaderFilename = ".license-header" // TODO: move to the plugin
	DefaultFontSize       = 12                // TODO: read from DPI settings from the monitor

	projectsFilename = "projects"
	settingsFilename = "settings"
)

var (
	App              = xdg.New("", "vidar")
	defaultConfigDir = App.ConfigHome()
	projects         *config
	settings         *config

	BuiltinFonts = map[string][]byte{
		"gomono":           gomono.TTF,
		"gomonobold":       gomonobold.TTF,
		"gomonoitalic":     gomonoitalic.TTF,
		"gomonobolditalic": gomonobolditalic.TTF,
	}
)

func init() {
	err := os.MkdirAll(defaultConfigDir, 0700)
	if err != nil {
		log.Printf("Error: Could not create config directory %s: %s", defaultConfigDir, err)
		return
	}
	projects, err = newConfig(projectsFilename, defaultConfigDir)
	if os.IsNotExist(err) {
		err = nil
	}
	if err != nil {
		log.Printf("Error reading projects: %s", err)
	}
	projects.setDefault("projects", []Project(nil))

	settings, err = newConfig(settingsFilename, defaultConfigDir)
	if os.IsNotExist(err) {
		err = nil
	}
	if err != nil {
		log.Printf("Error reading settings: %s", err)
	}
	settings.setDefault("fonts", []Font(nil))
}

type Font struct {
	Name string
	Size int
}

type Project struct {
	Name   string
	Path   string
	Gopath string
}

func (p Project) LicenseHeader() string {
	f, err := os.Open(filepath.Join(p.Path, LicenseHeaderFilename))
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		log.Printf("Error opening license header file: %s", err)
		return ""
	}
	defer f.Close()
	header, err := ioutil.ReadAll(f)
	if err != nil {
		log.Printf("Error reading license header file: %s", err)
		return ""
	}
	return string(header)
}

func (p Project) String() string {
	return p.Name
}

func (p Project) Environ() []string {
	environ := os.Environ()
	if p.Gopath == "" {
		return environ
	}
	environ = addEnv(environ, "GOPATH", p.Gopath, true)
	return addEnv(environ, "PATH", filepath.Join(p.Gopath, "bin"), false)
}

func addEnv(environ []string, key, value string, replace bool) []string {
	envKey := key + "="
	env := envKey + value
	for i, v := range environ {
		if !strings.HasPrefix(v, envKey) {
			continue
		}
		if !replace {
			env = fmt.Sprintf("%s%c%s", v, os.PathSeparator, value)
		}
		environ[i] = env
		return environ
	}
	return append(environ, env)
}

func Projects() (projs []Project) {
	projs, ok := projects.get("projects").([]Project)
	if !ok {
		return nil
	}
	return projs
}

func AddProject(project Project) {
	projects.set("projects", append(Projects(), project))
	if err := writeConfig(projects, projectsFilename); err != nil {
		log.Printf("Error updating projects file")
	}
}

func find(path, name string, extensions []string) (io.Reader, error) {
	d, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	i, err := d.Stat()
	if err != nil {
		return nil, err
	}
	if !i.IsDir() {
		return nil, fmt.Errorf("find called with a non-directory path")
	}
	infos, err := d.Readdir(-1)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for _, i := range infos {
		if i.IsDir() {
			r, err := find(filepath.Join(path, i.Name()), name, extensions)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				lastErr = err
				continue
			}
			return r, nil
		}
		for _, e := range extensions {
			if i.Name() != fmt.Sprintf("%s.%s", name, e) {
				continue
			}
			r, err := os.Open(filepath.Join(path, i.Name()))
			if err != nil {
				lastErr = err
				continue
			}
			return r, nil
		}
	}
	if lastErr != nil {
		return nil, err
	}
	return nil, os.ErrNotExist
}

func loadFont(font string) (io.Reader, error) {
	if b, ok := BuiltinFonts[font]; ok {
		return bytes.NewBuffer(b), nil
	}
	ext := []string{"ttf", "otf"}
	var lastErr error
	for _, p := range fontPaths {
		r, err := find(p, font, ext)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			lastErr = err
			continue
		}
		return r, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, os.ErrNotExist
}

// PrefFont returns the most preferred font found on the system.
func PrefFont(d gxui.Driver) gxui.Font {
	fonts, ok := settings.get("fonts").([]Font)
	if !ok {
		return parseDefaultFont(d)
	}
	for _, font := range fonts {
		r, err := loadFont(font.Name)
		if err != nil {
			log.Printf("Failed to load font %s: %s", font.Name, err)
			continue
		}
		f, err := parseFont(d, r, font.Size)
		if err != nil {
			log.Printf("Failed to parse font %s: %s", font.Name, err)
			continue
		}
		return f
	}
	return parseDefaultFont(d)
}

func parseDefaultFont(d gxui.Driver) gxui.Font {
	f, err := parseFont(d, bytes.NewBuffer(gomono.TTF), DefaultFontSize)
	if err != nil {
		// This is a well-tested font that should never fail to parse.
		panic(fmt.Errorf("failed to parse default font: %s", err))
	}
	return f
}

func parseFont(d gxui.Driver, f io.Reader, size int) (gxui.Font, error) {
	if c, ok := f.(io.Closer); ok {
		defer c.Close()
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	font, err := d.CreateFont(b, size)
	if err != nil {
		return nil, err
	}
	return font, nil
}

func writeConfig(cfg *config, fname string) error {
	f, err := os.Create(filepath.Join(defaultConfigDir, fname+".toml"))
	if err != nil {
		return fmt.Errorf("Could not create config file %s: %s", fname, err)
	}
	defer f.Close()
	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(cfg.data); err != nil {
		return fmt.Errorf("Could not marshal settings: %s", err)
	}
	return nil
}
