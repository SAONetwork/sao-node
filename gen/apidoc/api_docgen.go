package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sao-node/api"
	apitypes "sao-node/api/types"
	"sao-node/types"
	"sort"
	"strings"
	"unicode"

	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
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
	reflect.TypeOf(int64(42)):           int64(42),
	reflect.TypeOf(byte(7)):             byte(7),
	reflect.TypeOf([]byte{}):            []byte("byte array"),
	reflect.TypeOf(uint32(32)):          uint32(32),
	reflect.TypeOf(int32(32)):           int32(32),
	reflect.TypeOf(peer.ID("peer id")):  peer.ID("peer id"),
	reflect.TypeOf(int(32)):             int(32),
	reflect.TypeOf(true):                true,
}

func addExample(v interface{}) {
	ExampleValues[reflect.TypeOf(v)] = v
}

func init() {
	cid, _ := cid.Decode("bafkreihrwzskd3wixnkuikjidbx7ntgqugyiquglldl7yx2q2jbpzeoiyi")
	addExample(cid)

	addExample(apitypes.RenewResp{
		Results: map[string]string{
			"1e05407f-a7af-4b1c-b9e5-99d492f07720": "New Order=1",
			"1e05407f-a7af-4b1c-b9e5-99d492f07721": "renew fail root cause",
		},
	})

	sig := saotypes.JwsSignature{
		Protected: "eyJraWQiOiJkaWQ6c2lkOjY3YTJiZTczMTU3NDA4MjNlYmI2YTI3ZTJjZmQ3ODI1ZmMwMjEwMmE5NDIyMzVkZDI1ODlhZjQ3YTJkYWZiYTQ_dmVyc2lvbi1pZD02N2EyYmU3MzE1NzQwODIzZWJiNmEyN2UyY2ZkNzgyNWZjMDIxMDJhOTQyMjM1ZGQyNTg5YWY0N2EyZGFmYmE0IzhNalI1RlpCUUUiLCJhbGciOiJFUzI1NksifQ",
		Signature: "qbkzpCz_Yd8IeYmtmpGG2gdj-fkr5GwrHp5liBAOCSF5MQpHrZDFxp_GfTHv1sh8oDmR8JF2g9-GyVct7UJ24w",
	}
	addExample(sig)
	addExample(&types.MetadataProposal{
		Proposal: saotypes.QueryProposal{
			Owner:           "did:sid:67a2be7315740823ebb6a27e2cfd7825fc02102a942235dd2589af47a2dafba4",
			Keyword:         "fd248a7c-cf9f-4902-8327-58629aef96e9",
			GroupId:         "30293f0f-3e0f-4b3c-aff1-890a2fdf063b",
			KeywordType:     1,
			LastValidHeight: 711397,
			Gateway:         "/ip4/172.16.0.10/tcp/26660/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/127.0.0.1/tcp/26660/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/172.16.0.10/udp/26662/quic/webtransport/certhash/uEiCzHFKwct72TeBBh7-LUQ8L9QWwAo0b7d4VvsatjsQlQQ/certhash/uEiBKclz2BT5PNmQ9LIZr0DdhY7MpLLNXz8xLVdzSGyVXbA/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/127.0.0.1/udp/26662/quic/webtransport/certhash/uEiCzHFKwct72TeBBh7-LUQ8L9QWwAo0b7d4VvsatjsQlQQ/certhash/uEiBKclz2BT5PNmQ9LIZr0DdhY7MpLLNXz8xLVdzSGyVXbA/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT",
			CommitId:        "",
			Version:         "",
		},
		JwsSignature: sig,
	})

	loadResp := apitypes.LoadResp{
		DataId:   "fd248a7c-cf9f-4902-8327-58629aef96e9",
		Alias:    "note_ca0b1124-f013-4c69-8249-41694d540871",
		CommitId: "fd248a7c-cf9f-4902-8327-58629aef96e9",
		Version:  "v0",
		Cid:      "bafkreide7eax3pd3qsbolguprfta7thinb4wmbvyh2kestrdeiydg77tsq",
		Content:  `{"content":"","isEdit":false,"time":"2022-12-20 06:41","title":"sample"}`,
	}
	addExample(loadResp)

	addExample(&types.OrderStoreProposal{
		Proposal: saotypes.Proposal{
			Owner:         "did:sid:67a2be7315740823ebb6a27e2cfd7825fc02102a942235dd2589af47a2dafba4",
			Provider:      "cosmos197vlml2yg75rg9dmf07sau0mn0053p9dscrfsf",
			GroupId:       "30293f0f-3e0f-4b3c-aff1-890a2fdf063b",
			Duration:      31536000,
			Replica:       1,
			Timeout:       86400,
			Alias:         "notes",
			DataId:        "c2b37317-9612-41fe-8260-7c8aea0dbd07",
			CommitId:      "c2b37317-9612-41fe-8260-7c8aea0dbd07",
			Tags:          nil,
			Cid:           "bafkreib3yoebpagjbkvhrsyhi7jpllylcqt4zpime5vho6ehpljv3dda4u",
			Rule:          "",
			ExtendInfo:    "",
			Size_:         40,
			Operation:     1,
			ReadonlyDids:  []string{},
			ReadwriteDids: []string{},
		},
		JwsSignature: sig,
	})

	addExample(apitypes.CreateResp{
		DataId: "c2b37317-9612-41fe-8260-7c8aea0dbd07",
		Alias:  "notes",
		TxId:   "",
		Cid:    "bafkreib3yoebpagjbkvhrsyhi7jpllylcqt4zpime5vho6ehpljv3dda4u",
	})

	addExample(&types.OrderTerminateProposal{
		Proposal: saotypes.TerminateProposal{
			Owner:  "did:sid:67a2be7315740823ebb6a27e2cfd7825fc02102a942235dd2589af47a2dafba4",
			DataId: "fd248a7c-cf9f-4902-8327-58629aef96e9",
		},
		JwsSignature: sig,
	})

	addExample(apitypes.DeleteResp{
		DataId: "fd248a7c-cf9f-4902-8327-58629aef96e9",
		Alias:  "note_ca0b1124-f013-4c69-8249-41694d540871",
	})

	addExample(apitypes.UpdateResp{
		DataId:   "fd248a7c-cf9f-4902-8327-58629aef96e9",
		CommitId: "fd248a7c-cf9f-4902-8327-58629aef96e9",
		Alias:    "notes",
		TxId:     "",
		Cid:      "bafkreide7eax3pd3qsbolguprfta7thinb4wmbvyh2kestrdeiydg77tsq",
	})

	addExample(apitypes.ShowCommitsResp{
		DataId:  "c2b37317-9612-41fe-8260-7c8aea0dbd07",
		Alias:   "notes",
		Commits: []string{"c2b37317-9612-41fe-8260-7c8aea0dbd07711196", "85de5f5e-0cfb-4e0c-abe7-bf93aec087f3712565"},
	})

	addExample(types.PeerInfo{
		ID:    peer.ID("12D3KooWSsWdkkKzvHSV6cc8ET6eyiDoHkYeW9GwF9RuQXiWF3cS"),
		Addrs: []string{"/ip4/127.0.0.1/tcp/26660", "/ip4/172.16.0.11/tcp/26660"},
	})

	addExample(apitypes.GetPeerInfoResp{
		PeerInfo: "/ip4/172.16.0.10/tcp/26660/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/127.0.0.1/tcp/26660/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/172.16.0.10/udp/26662/quic/webtransport/certhash/uEiCzHFKwct72TeBBh7-LUQ8L9QWwAo0b7d4VvsatjsQlQQ/certhash/uEiBKclz2BT5PNmQ9LIZr0DdhY7MpLLNXz8xLVdzSGyVXbA/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/127.0.0.1/udp/26662/quic/webtransport/certhash/uEiCzHFKwct72TeBBh7-LUQ8L9QWwAo0b7d4VvsatjsQlQQ/certhash/uEiBKclz2BT5PNmQ9LIZr0DdhY7MpLLNXz8xLVdzSGyVXbA/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT",
	})

	addExample(apitypes.GenerateTokenResp{
		Server: "localhost:5152",
		Token:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJrZXkiOiJkaWQ6a2V5OnpRM3NodXZYcWZMTHFDbmtHaGh5VkdMQ3EyOXR1bktURmVINjdla2QzVHlyMmVaWFgiLCJleHAiOjE2NzE2NzgzMzF9.jV6Jk4UQnl8TfXS9WtjYw2JXMKaIeAulNwQma_fQVAs",
	})

	addExample(apitypes.GetUrlResp{
		Url: "http://localhost:5152/saonetwork/a4cc25ff-80b1-4815-8c5e-af3ff133420b",
	})

	addExample(saotypes.PermissionProposal{
		Owner:         "did:key:zQ3shuvXqfLLqCnkGhhyVGLCq29tunKTFeH67ekd3Tyr2eZXX",
		DataId:        "4821b0f9-736c-4d48-95b7-4f80cd432781",
		ReadonlyDids:  []string{"did:key:zQ3shpp99D7y2z3B2Qq6yGpWcTrxLHHnawrdHDXhVFjhE8x6h"},
		ReadwriteDids: []string{"did:key:zQ3shpp99D7y2z3B2Qq6yGpWcTrxLHHnawrdHDXhVFjhE8x66"},
	})

	addExample(apitypes.UpdatePermissionResp{DataId: "4821b0f9-736c-4d48-95b7-4f80cd432781"})

	addExample(saotypes.RenewProposal{
		Owner:    "did:key:zQ3shuvXqfLLqCnkGhhyVGLCq29tunKTFeH67ekd3Tyr2eZXX",
		Duration: 31536000,
		Timeout:  86400,
		Data:     []string{"4821b0f9-736c-4d48-95b7-4f80cd432781"},
	})

	addExample([]types.MigrateInfo{{
		DataId:           "4821b0f9-736c-4d48-95b7-4f80cd432781",
		OrderId:          0,
		Cid:              "bafkreide7eax3pd3qsbolguprfta7thinb4wmbvyh2kestrdeiydg77tsq",
		FromProvider:     "did:sid:67a2be7315740823ebb6a27e2cfd7825fc02102a942235dd2589af47a2dafba4",
		ToProvider:       "did:sid:67a2be7315740823ebb6a27e2cfd7825fc02102a942235dd2589af47a2dafba5",
		MigrateTxHash:    "",
		MigrateTxHeight:  1,
		CompleteTxHash:   "",
		CompleteTxHeight: 1,
		State:            types.MigrateStateComplete,
	}})

	addExample(apitypes.MigrateResp{
		TxHash: "",
		Results: map[string]string{
			"4821b0f9-736c-4d48-95b7-4f80cd432781": "SUCCESS",
		},
	})

	addExample(types.OrderInfo{
		DataId:    "4821b0f9-736c-4d48-95b7-4f80cd432781",
		Owner:     "did:sid:67a2be7315740823ebb6a27e2cfd7825fc02102a942235dd2589af47a2dafba4",
		Cid:       cid,
		StagePath: "~/.saonode/staging",

		State:   types.OrderStateStaged,
		LastErr: "",
	})

	addExample(types.ShardInfo{
		OrderId:        1,
		DataId:         "4821b0f9-736c-4d48-95b7-4f80cd432781",
		Cid:            cid,
		Owner:          "did:sid:67a2be7315740823ebb6a27e2cfd7825fc02102a942235dd2589af47a2dafba4",
		Gateway:        "cosmos197vlml2yg75rg9dmf07sau0mn0053p9dscrfsf",
		OrderOperation: "1",
		ShardOperation: "1",
		CompleteHash:   "",
		CompleteHeight: 1,
		Size:           1,
		State:          types.ShardStateTxSent,
		LastErr:        "",
	})
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
