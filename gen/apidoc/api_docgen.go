package main

import (
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/libp2p/go-libp2p/core/peer"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sao-node/api"
	"sort"
	"strings"
	"unicode"
)

func main() {
	comments, groupComments := ParseApiASTInfo(os.Args[1], os.Args[2], os.Args[3], os.Args[4])
	_, t, permStruct := GetAPIType(os.Args[2], os.Args[3])
	groups := make(map[string]*MethodGroup)
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)

		groupName := MethodGroupFromName(m.Name)

		// hack for common group
		if _, exists := groupComments[groupName]; !exists {
			groupName = "Common"
		}

		g, ok := groups[groupName]
		if !ok {
			g = new(MethodGroup)
			g.Header = groupComments[groupName]
			g.GroupName = groupName
			groups[groupName] = g
		}

		var args []interface{}
		ft := m.Func.Type()
		for j := 2; j < ft.NumIn(); j++ {
			inp := ft.In(j)
			args = append(args, ExampleValue(m.Name, inp, nil))
		}

		v, err := json.MarshalIndent(args, "", "  ")
		if err != nil {
			panic(err)
		}

		outv := ExampleValue(m.Name, ft.Out(0), nil)
		ov, err := json.MarshalIndent(outv, "", "  ")
		if err != nil {
			panic(err)
		}

		g.Methods = append(g.Methods, &Method{
			Name:            m.Name,
			Comment:         comments[m.Name],
			InputExample:    string(v),
			ResponseExample: string(ov),
		})
	}

	var groupslice []*MethodGroup
	for _, g := range groups {
		groupslice = append(groupslice, g)
	}

	sort.Slice(groupslice, func(i, j int) bool {
		return groupslice[i].GroupName < groupslice[j].GroupName
	})

	fmt.Printf("# Groups\n")

	for _, g := range groupslice {
		fmt.Printf("* [%s](#%s)\n", g.GroupName, g.GroupName)
		for _, method := range g.Methods {
			fmt.Printf("  * [%s](#%s)\n", method.Name, method.Name)
		}
	}

	for _, g := range groupslice {
		g := g
		fmt.Printf("## %s\n", g.GroupName)
		fmt.Printf("%s\n\n", g.Header)

		sort.Slice(g.Methods, func(i, j int) bool {
			return g.Methods[i].Name < g.Methods[j].Name
		})

		for _, m := range g.Methods {
			fmt.Printf("### %s\n", m.Name)
			fmt.Printf("%s\n\n", m.Comment)

			var meth reflect.StructField
			var ok bool
			for _, ps := range permStruct {
				meth, ok = ps.FieldByName(m.Name)
				if ok {
					break
				}
			}
			if !ok {
				panic("no perms for method: " + m.Name)
			}

			perms := meth.Tag.Get("perm")

			fmt.Printf("Perms: %s\n\n", perms)

			if strings.Count(m.InputExample, "\n") > 0 {
				fmt.Printf("Inputs:\n```json\n%s\n```\n\n", m.InputExample)
			} else {
				fmt.Printf("Inputs: `%s`\n\n", m.InputExample)
			}

			if strings.Count(m.ResponseExample, "\n") > 0 {
				fmt.Printf("Response:\n```json\n%s\n```\n\n", m.ResponseExample)
			} else {
				fmt.Printf("Response: `%s`\n\n", m.ResponseExample)
			}
		}
	}
}

func GetAPIType(name, pkg string) (i interface{}, t reflect.Type, permStruct []reflect.Type) {
	switch pkg {
	case "api":
		switch name {
		case "SaoApi":
			i = &api.SaoApiStruct{}
			t = reflect.TypeOf(new(struct{ api.SaoApi })).Elem()
			permStruct = append(permStruct, reflect.TypeOf(api.SaoApiStruct{}.Internal))
		}
	}
	return
}

var ExampleValues = map[reflect.Type]interface{}{
	reflect.TypeOf(auth.Permission("")): auth.Permission("write"),
	reflect.TypeOf(""):                  "string value",
	reflect.TypeOf(uint64(42)):          uint64(42),
	reflect.TypeOf(byte(7)):             byte(7),
	reflect.TypeOf([]byte{}):            []byte("byte array"),
	reflect.TypeOf(uint32(32)):          uint32(32),
	reflect.TypeOf(int32(32)):           int32(32),
	reflect.TypeOf(peer.ID("peer id")):  peer.ID("peer id"),
	reflect.TypeOf(int(32)):             int(32),
}

func ExampleValue(method string, t, parent reflect.Type) interface{} {
	v, ok := ExampleValues[t]
	if ok {
		return v
	}

	switch t.Kind() {
	case reflect.Slice:
		out := reflect.New(t).Elem()
		out = reflect.Append(out, reflect.ValueOf(ExampleValue(method, t.Elem(), t)))
		return out.Interface()
	case reflect.Chan:
		return ExampleValue(method, t.Elem(), nil)
	case reflect.Struct:
		es := exampleStruct(method, t, parent)
		v := reflect.ValueOf(es).Elem().Interface()
		ExampleValues[t] = v
		return v
	case reflect.Array:
		out := reflect.New(t).Elem()
		for i := 0; i < t.Len(); i++ {
			out.Index(i).Set(reflect.ValueOf(ExampleValue(method, t.Elem(), t)))
		}
		return out.Interface()

	case reflect.Ptr:
		if t.Elem().Kind() == reflect.Struct {
			es := exampleStruct(method, t.Elem(), t)
			//ExampleValues[t] = es
			return es
		}
	case reflect.Interface:
		return struct{}{}
	}

	panic(fmt.Sprintf("No example value for type: %s (method '%s')", t, method))
}

func exampleStruct(method string, t, parent reflect.Type) interface{} {
	ns := reflect.New(t)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Type == parent {
			continue
		}
		if strings.Title(f.Name) == f.Name {
			ns.Elem().Field(i).Set(reflect.ValueOf(ExampleValue(method, f.Type, t)))
		}
	}

	return ns.Interface()
}

func MethodGroupFromName(mn string) string {
	i := strings.IndexFunc(mn[1:], func(r rune) bool {
		return unicode.IsUpper(r)
	})
	if i < 0 {
		return ""
	}
	return mn[:i+1]
}

type MethodGroup struct {
	GroupName string
	Header    string
	Methods   []*Method
}

type Method struct {
	Comment         string
	Name            string
	InputExample    string
	ResponseExample string
}

func ParseApiASTInfo(apiFile, iface, pkg, dir string) (comments map[string]string, groupDocs map[string]string) { //nolint:golint
	fset := token.NewFileSet()
	apiDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Println("./api filepath absolute error: ", err)
		return
	}
	apiFile, err = filepath.Abs(apiFile)
	if err != nil {
		fmt.Println("filepath absolute error: ", err, "file:", apiFile)
		return
	}
	pkgs, err := parser.ParseDir(fset, apiDir, nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		fmt.Println("parse error: ", err)
		return
	}

	ap := pkgs[pkg]

	f := ap.Files[apiFile]

	cmap := ast.NewCommentMap(fset, f, f.Comments)

	v := &Visitor{iface, make(map[string]ast.Node)}
	ast.Walk(v, ap)

	comments = make(map[string]string)
	groupDocs = make(map[string]string)
	for mn, node := range v.Methods {
		filteredComments := cmap.Filter(node).Comments()
		if len(filteredComments) == 0 {
			comments[mn] = NoComment
		} else {
			for _, c := range filteredComments {
				if strings.HasPrefix(c.Text(), "MethodGroup:") {
					parts := strings.Split(c.Text(), "\n")
					groupName := strings.TrimSpace(parts[0][12:])
					comment := strings.Join(parts[1:], "\n")
					groupDocs[groupName] = comment

					break
				}
			}

			l := len(filteredComments) - 1
			if len(filteredComments) > 1 {
				l = len(filteredComments) - 2
			}
			last := filteredComments[l].Text()
			if !strings.HasPrefix(last, "MethodGroup:") {
				comments[mn] = last
			} else {
				comments[mn] = NoComment
			}
		}
	}
	return comments, groupDocs
}

const NoComment = "There are not yet any comments for this method."

type Visitor struct {
	Root    string
	Methods map[string]ast.Node
}

func (v *Visitor) Visit(node ast.Node) ast.Visitor {
	st, ok := node.(*ast.TypeSpec)
	if !ok {
		return v
	}

	if st.Name.Name != v.Root {
		return nil
	}

	iface := st.Type.(*ast.InterfaceType)
	for _, m := range iface.Methods.List {
		if len(m.Names) > 0 {
			v.Methods[m.Names[0].Name] = m
		}
	}

	return v
}
