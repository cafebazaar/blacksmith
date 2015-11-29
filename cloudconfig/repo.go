package cloudconfig // import "github.com/cafebazaar/aghajoon/cloudconfig"

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"sync"
	"text/template"
	"text/template/parse"
)

type Repo struct {
	templates   *template.Template
	dataSources map[string]DataSource
	executeLock sync.Mutex
}

func findFiles(path string) ([]string, error) {
	infos, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0)
	for i := range infos {
		if !infos[i].IsDir() && infos[i].Name()[0] != '.' {
			files = append(files, infos[i].Name())
		}
	}
	return files, nil
}

func parseFile(filename string) ([]*parse.Tree, error) {
	_ = fmt.Errorf
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	trees := make([]*parse.Tree, 0)
	name := strings.Split(path.Base(filename), ".")[0]
	// place holder hack
	// parse needs to know funcs before execution and Value() is context sensitive
	// so we cannot know its arguments before requests
	treeSet, err := parse.Parse(name, string(data), "<<", ">>",
		map[string]interface{}{
			"V": func(key string) (interface{}, error) {
				return "FUNC PLACEHOLDER", nil
			},
			"S": func(key string, value string) (interface{}, error) {
				return "FUNC PLACEHOLDER", nil
			},
			"D": func(key string) (interface{}, error) {
				return "FUNC PLACEHOLDER", nil
			},
			"VD": func(key string) (interface{}, error) {
				return "FUNC PLACEHOLDER", nil
			},
			"b64": func(text string) interface{} {
				return "FUNC PLACEHOLDER"
			},
			"b64template": func(text string) (interface{}, error) {
				return "FUNC PLACEHOLDER", nil
			},
		})
	if err != nil {
		return nil, err
	}
	for _, tree := range treeSet {
		trees = append(trees, tree)
	}
	return trees, nil
}

func FromPath(dataSources map[string]DataSource, tmplPath string) (*Repo, error) {
	files, err := findFiles(tmplPath)
	if err != nil {
		return nil, err
	}

	t := template.New("")
	for i := range files {
		trees, err := parseFile(path.Join(tmplPath, files[i]))
		if err != nil {
			return nil, err
		}
		for i := range trees {
			t.AddParseTree(trees[i].Name, trees[i])
		}
	}
	return &Repo{
		templates:   t,
		dataSources: dataSources,
	}, nil
}

func (r *Repo) ExecuteTemplate(templateName string, c *ConfigContext) (string, error) {
	// rewrite funcs to include context and hold a lock so it doesn't get overwrite
	buf := new(bytes.Buffer)
	r.templates.Funcs(map[string]interface{}{
		"V": func(key string) (interface{}, error) {
			return GetValue(r.dataSources, c, key)
		},
		"S": func(key string, value string) (interface{}, error) {
			return Set(r.dataSources, c, key, value)
		},
		"VD": func(key string) (interface{}, error) {
			return GetAndDelete(r.dataSources, c, key)
		},
		"D": func(key string) (interface{}, error) {
			return Delete(r.dataSources, c, key)
		},
		"b64": func(text string) interface{} {
			return base64.StdEncoding.EncodeToString([]byte(text))
		},
		"b64template": func(templateName string) (interface{}, error) {
			text, err := r.ExecuteTemplate(templateName, c)
			if err != nil {
				return nil, err
			}
			return base64.StdEncoding.EncodeToString([]byte(text)), nil
		},
	})
	err := r.templates.ExecuteTemplate(buf, templateName, c.Map())
	return buf.String(), err
}

func (r *Repo) GenerateConfig(c *ConfigContext) (string, error) {
	r.executeLock.Lock()
	defer r.executeLock.Unlock()
	if r.templates.Lookup("main") == nil {
		return "", nil
	}
	return r.ExecuteTemplate("main", c)
}
