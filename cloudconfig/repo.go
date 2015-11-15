package cloudconfig

import (
	"fmt"
	"io"
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
		if !infos[i].IsDir() && strings.HasSuffix(infos[i].Name(), ".yaml") {
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
		map[string]interface{}{"V": func(key string) (interface{}, error) {
			return "FUNC PLACEHOLDER", nil
		}})
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

func (r *Repo) GenerateConfig(w io.Writer, c *ConfigContext) error {
	// rewrite funcs to include context and hold a lock so it doesn't get overwrite
	r.executeLock.Lock()
	defer r.executeLock.Unlock()
	r.templates.Funcs(map[string]interface{}{"V": func(key string) (interface{}, error) {
		return Value(r.dataSources, c, key)
	}})
	r.templates.ExecuteTemplate(w, "main", c.Map())
	return nil
}
