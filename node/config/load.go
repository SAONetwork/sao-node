package config

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sao-node/types"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/kelseyhightower/envconfig"
)

func ConfigComment(t interface{}) ([]byte, error) {
	return ConfigUpdate(t, nil, true)
}
func ConfigUpdate(cfgCur, cfgDef interface{}, comment bool) ([]byte, error) {
	var nodeStr, defStr string
	if cfgDef != nil {
		buf := new(bytes.Buffer)
		e := toml.NewEncoder(buf)
		if err := e.Encode(cfgDef); err != nil {
			return nil, types.Wrap(types.ErrEncodeConfigFailed, err)
		}

		defStr = buf.String()
	}

	{
		buf := new(bytes.Buffer)
		e := toml.NewEncoder(buf)
		if err := e.Encode(cfgCur); err != nil {
			return nil, types.Wrap(types.ErrEncodeConfigFailed, err)
		}

		nodeStr = buf.String()
	}

	if comment {
		// create a map of default lines, so we can comment those out later
		defLines := strings.Split(defStr, "\n")
		defaults := map[string]struct{}{}
		for i := range defLines {
			l := strings.TrimSpace(defLines[i])
			if len(l) == 0 {
				continue
			}
			if l[0] == '#' || l[0] == '[' {
				continue
			}
			defaults[l] = struct{}{}
		}

		nodeLines := strings.Split(nodeStr, "\n")
		var outLines []string

		sectionRx := regexp.MustCompile(`\[(.+)]`)
		var section string

		for i, line := range nodeLines {
			// if this is a section, track it
			trimmed := strings.TrimSpace(line)
			if len(trimmed) > 0 {
				if trimmed[0] == '[' {
					m := sectionRx.FindSubmatch([]byte(trimmed))
					if len(m) != 2 {
						return nil, types.Wrapf(types.ErrInvalidConfig, "section didn't match (line %d)", i)
					}
					section = string(m[1])

					// never comment sections
					outLines = append(outLines, line)
					continue
				}
			}

			pad := strings.Repeat(" ", len(line)-len(strings.TrimLeftFunc(line, unicode.IsSpace)))

			// see if we have docs for this field
			{
				lf := strings.Fields(line)
				if len(lf) > 1 {
					doc := findDoc(cfgCur, section, lf[0])

					if doc != nil {
						// found docfield, emit clidoc comment
						if len(doc.Comment) > 0 {
							for _, docLine := range strings.Split(doc.Comment, "\n") {
								outLines = append(outLines, pad+"# "+docLine)
							}
							outLines = append(outLines, pad+"#")
						}

						outLines = append(outLines, pad+"# type: "+doc.Type)
					}

					//outLines = append(outLines, pad+"# env var: SAO_"+strings.ToUpper(strings.ReplaceAll(section, ".", "_"))+"_"+strings.ToUpper(lf[0]))
				}
			}

			// if there is the same line in the default config, comment it out it output
			if _, found := defaults[strings.TrimSpace(nodeLines[i])]; (cfgDef == nil || found) && len(line) > 0 {
				line = pad + "#" + line[len(pad):]
			}
			outLines = append(outLines, line)
			if len(line) > 0 {
				outLines = append(outLines, "")
			}
		}

		nodeStr = strings.Join(outLines, "\n")
	}

	// sanity-check that the updated config parses the same way as the current one
	if cfgDef != nil {
		cfgUpdated, err := FromReader(strings.NewReader(nodeStr), cfgDef)
		if err != nil {
			return nil, types.Wrap(types.ErrDecodeConfigFailed, err)
		}

		if !reflect.DeepEqual(cfgCur, cfgUpdated) {
			return nil, types.Wrapf(types.ErrInvalidConfig, "updated config didn't match current config")
		}
	}

	return []byte(nodeStr), nil
}
func findDoc(root interface{}, section, name string) *DocField {
	rt := fmt.Sprintf("%T", root)[len("*config."):]

	doc := findDocSect(rt, section, name)
	if doc != nil {
		return doc
	}

	return findDocSect("Common", section, name)
}

func findDocSect(root, section, name string) *DocField {
	path := strings.Split(section, ".")

	docSection := Doc[root]
	for _, e := range path {
		if docSection == nil {
			return nil
		}

		for _, field := range docSection {
			if field.Name == e {
				docSection = Doc[field.Type]
				break
			}

		}
	}

	for _, df := range docSection {
		if df.Name == name {
			return &df
		}
	}

	return nil
}

// FromReader loads config from a reader instance.
func FromReader(reader io.Reader, def interface{}) (interface{}, error) {
	cfg := def
	_, err := toml.NewDecoder(reader).Decode(cfg)
	if err != nil {
		return nil, err
	}

	err = envconfig.Process("SAO", cfg)
	if err != nil {
		return nil, types.Wrapf(types.ErrInvalidConfig, "processing env vars overrides: %w", err)
	}

	return cfg, nil
}
