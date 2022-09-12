package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/epicbytes/protocommon/common"
)

const (
	contextPackage = protogen.GoImportPath("context")
	fmtPackage     = protogen.GoImportPath("fmt")
)

func main() {
	var flags flag.FlagSet
	protogen.Options{
		ParamFunc: flags.Set,
	}.Run(func(gen *protogen.Plugin) error {
		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			generateHelpers(gen, f)
		}
		return nil
	})
}

func getFieldsFromMessage(messages []*protogen.Message, messageName string) []*protogen.Field {
	for _, message := range messages {
		if string(message.Desc.FullName().Name()) == messageName {
			return message.Fields
		}
	}
	return nil
}

func getFieldFromMessage(messages []*protogen.Message, entityName string, fieldName string) *protogen.Field {
	for _, message := range messages {
		if string(message.Desc.FullName().Name()) == entityName {
			for _, field := range message.Fields {
				if field.GoName == fieldName {
					return field
				}
			}
		}
	}
	return nil
}

func hasBodyParams(fields []*protogen.Field) bool {
	for _, field := range fields {
		if strings.Contains(field.Comments.Leading.String(), "In: body") {
			return true
		}
	}

	return false
}

var (
	slashSlash = []byte("//")
	moduleStr  = []byte("module")
)

// ModulePath returns the module path from the gomod file text.
// If it cannot find a module path, it returns an empty string.
// It is tolerant of unrelated problems in the go.mod file.
func ModulePath(mod []byte) string {
	for len(mod) > 0 {
		line := mod
		mod = nil
		if i := bytes.IndexByte(line, '\n'); i >= 0 {
			line, mod = line[:i], line[i+1:]
		}
		if i := bytes.Index(line, slashSlash); i >= 0 {
			line = line[:i]
		}
		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, moduleStr) {
			continue
		}
		line = line[len(moduleStr):]
		n := len(line)
		line = bytes.TrimSpace(line)
		if len(line) == n || len(line) == 0 {
			continue
		}

		if line[0] == '"' || line[0] == '`' {
			p, err := strconv.Unquote(string(line))
			if err != nil {
				return "" // malformed quoted string or multiline module path
			}
			return p
		}

		return string(line)
	}
	return "" // missing module path
}

func generateHelpers(gen *protogen.Plugin, file *protogen.File) *protogen.GeneratedFile {
	filename := file.GeneratedFilenamePrefix + "_helpers.pb.go"

	var pwd = os.Getenv("PWD")
	moduleFile, err := os.ReadFile(fmt.Sprintf("%s/../../go.mod", pwd))
	if err != nil {

	}
	var path = ModulePath(moduleFile)
	var keeperPackage = protogen.GoImportPath(fmt.Sprintf("%s/internal/keeper", path))
	var commonPackage = protogen.GoImportPath(fmt.Sprintf("%s/pkg/common", path))

	g := gen.NewGeneratedFile(filename, file.GoImportPath)
	g.P("// Code generated by protoc-gen-go-helpers. DO NOT EDIT.")
	g.P()
	g.P("package ", file.GoPackageName)
	g.P()
	for _, msg := range file.Messages {

		const stringType = "string"
		const uint32Type = "uint32"

		/*if strings.Contains(string(msg.Comments.Trailing), "@feature:\"keeper=") {

			re := regexp.MustCompile(`(?m)@feature:"keeper=(.*)"`)
			for _, match := range re.FindAllStringSubmatch(string(msg.Comments.Trailing), -1) {

				g.P("func (x *", msg.GoIdent, ") EncryptFields(ctx ", g.QualifiedGoIdent(contextPackage.Ident("Context")), ", keepr ", g.QualifiedGoIdent(keeperPackage.Ident("Keeper")), ") {")
				g.P("keepr.TransitEncrypt(ctx, x, \"", match[1], "\")")
				g.P("}")

				g.P("func (x *", msg.GoIdent, ") DecryptFields(ctx ", g.QualifiedGoIdent(contextPackage.Ident("Context")), ", keepr ", g.QualifiedGoIdent(keeperPackage.Ident("Keeper")), ") {")
				g.P("keepr.TransitDecrypt(ctx, x, \"", match[1], "\")")
				g.P("}")
			}
		}*/

		if strings.Contains(string(msg.Comments.Trailing), "@parser:\"list\"") {
			g.P("func (x *", msg.GoIdent, ")  GetFilter() ", g.QualifiedGoIdent(protogen.GoIdent{GoName: "M", GoImportPath: "go.mongodb.org/mongo-driver/bson"}), " {")
			g.P("query := bson.M{}")

			for _, fld := range msg.Fields {

				if strings.Contains(string(fld.Comments.Leading), "@parser:\"filter\"") {
					switch fld.Desc.Kind().String() {
					case stringType:
						g.P("if x.", fld.GoName, " != \"\" {")
					case uint32Type:
						g.P("if x.", fld.GoName, " != 0 {")
					default:
						g.P("if x.", fld.GoName, " != \"\" {")

					}
					g.P("query[\"", fld.Desc.Name(), "\"] = x.", fld.GoName)
					g.P("}")
				}

			}

			g.P("return query")
			g.P("}")
			g.P()
			g.P("func (x *", msg.GoIdent, ") GetOptions() *", g.QualifiedGoIdent(protogen.GoIdent{GoName: "FindOptions", GoImportPath: "go.mongodb.org/mongo-driver/mongo/options"}), " {")
			g.P("var Options = &", g.QualifiedGoIdent(protogen.GoIdent{GoName: "FindOptions", GoImportPath: "go.mongodb.org/mongo-driver/mongo/options"}), "{}")
			if strings.Contains(string(msg.Comments.Trailing), "paging:true") {
				g.P("var limit int64 = 20")
				g.P("if x.Limit > 0 {")
				g.P("limit = x.Limit")
				g.P("}")
				g.P("Options.SetLimit(limit)")
				g.P("Options.SetSkip(x.Skip)")
				g.P("Options.SetSort(bson.M{\"_id\": 1})")
			}
			g.P("return Options")
			g.P("}")
		}
		/*if strings.Contains(string(msg.Comments.Trailing), "@parser:\"swag\"") {
			g.P(fmt.Sprintf("// swagger:parameters %sWrapper", Camel(msg.GoIdent.GoName)))
			g.P("// ", msg.GoIdent.GoName, "Wrapper wrapper for ", msg.GoIdent.GoName)
			g.P("type ", msg.GoIdent.GoName, "Wrapper struct {")
			g.P("// In: body")
			g.P("Body ", msg.GoIdent.GoName)
			g.P("}")
		}*/
		if strings.Contains(string(msg.Comments.Trailing), "@parser:\"fiber\"") {
			g.P("func (x *", msg.GoIdent, ") BindFromFiber(ctx *", g.QualifiedGoIdent(protogen.GoIdent{GoName: "Ctx", GoImportPath: "github.com/gofiber/fiber/v2"}), ") error {")
			if len(msg.Fields) > 0 {
				g.P("err := ctx.QueryParser(x)")
				g.P("if err != nil {")
				g.P("return err")
				g.P("}")
				if hasBodyParams(msg.Fields) {
					g.P("err = ctx.BodyParser(x)")
					g.P("if err != nil {")
					g.P("return err")
					g.P("}")
				}
			}
			for _, field := range msg.Fields {

				options2 := field.Desc.Options().(*descriptorpb.FieldOptions)
				ext2 := proto.GetExtension(options2, common.E_FieldOption).(*common.ModelFieldOption)
				if ext2 != nil {
					g.P("// In:", *ext2.Source)
				}
				if strings.Contains(string(field.Comments.Leading), "In: context") {
					ind := ""
					if field.Desc.IsList() {
						ind = "[]"
					}
					g.P("if ctx.Locals(\"", field.Desc.Name(), "\") != nil {")
					g.P("x.", field.GoName, " = ctx.Locals(\"", field.Desc.Name(), "\").(", ind, field.Desc.Kind().String(), ")")
					g.P("}")
				}
				if strings.Contains(string(field.Comments.Leading), "In: path") {
					snakedFieldName := Snake(field.GoName)
					switch field.Desc.Kind().String() {
					case uint32Type:
						g.P(snakedFieldName, ", err := ctx.ParamsInt(\"", snakedFieldName, "\")")
						g.P("if err != nil {")
						g.P("return err")
						g.P("}")
						g.P("x.", field.GoName, " = uint32(", snakedFieldName, ")")
					default:
						g.P("ERROR NON PARSABLE TYPE ", field.Desc.Kind().String())
					}
				}
			}
			g.P("return nil")
			g.P("}")
		}

		if strings.Contains(string(msg.Comments.Trailing), "@pickFromArrayWPagination:") {
			re := regexp.MustCompile(`(?m)@pickFromArrayWPagination:"(.*)"`)
			for _, match := range re.FindAllStringSubmatch(string(msg.Comments.Trailing), -1) {
				modelForMerge := match[1]

				g.P()

				g.P("func (x *", msg.GoIdent, ") PickFrom", modelForMerge, "(request []*", modelForMerge, ", pagination *", g.QualifiedGoIdent(commonPackage.Ident("Pagination")), ") {")

				for _, field := range msg.Fields {
					switch field.GoName {
					case "Items":
						g.P("if request == nil { return }")
						g.P("var items =  make([]*", string(field.Desc.Message().Name()), ", 0)")
						requestFields := getFieldsFromMessage(file.Messages, string(field.Desc.Message().Name()))
						g.P("if len(request) > 0 {")
						g.P("for _, req := range request{")
						g.P("var item = new(", string(field.Desc.Message().Name()), ")")
						for _, requestField := range requestFields {
							typeFromField := requestField.Desc.Kind().String()
							if requestField.Desc.IsList() {
								typeFromField = fmt.Sprintf("[]%s", typeFromField)
							}
							switch typeFromField {
							case stringType:
								g.P("if req.", requestField.GoName, " != \"\" {")
								g.P("item.", requestField.GoName, " = req.Get", requestField.GoName, "()")
								g.P("}")
							case uint32Type:
								g.P("if req.", requestField.GoName, " != 0 {")
								g.P("item.", requestField.GoName, " = req.Get", requestField.GoName, "()")
								g.P("}")
							case "[]string", "[]uint32":
								g.P("if len(req.", requestField.GoName, ") > 0 {")
								g.P("item.", requestField.GoName, " = req.Get", requestField.GoName, "()")
								g.P("}")
							default:
								g.P("item.", requestField.GoName, " = req.Get", requestField.GoName, "()")
							}
						}
						g.P("items = append(items, item)")
						g.P("}")
						g.P("}")
						g.P("x.Items = items")
					case "Pagination":
						g.P("x.Pagination = pagination")
					}
				}

				g.P("}")
			}
		}
		if strings.Contains(string(msg.Comments.Trailing), "@pickFrom:") {
			re := regexp.MustCompile(`(?m)@pickFrom:"(.*)"`)
			for _, match := range re.FindAllStringSubmatch(string(msg.Comments.Trailing), -1) {
				modelForMerge := match[1]

				requestFields := getFieldsFromMessage(file.Messages, msg.GoIdent.GoName)
				g.P()

				g.P("func (x *", msg.GoIdent, ") PickFrom", modelForMerge, "(request *", modelForMerge, ") {")
				g.P("if request == nil { return }")
				for _, requestField := range requestFields {
					entityField := getFieldFromMessage(file.Messages, modelForMerge, requestField.GoName)
					if entityField == nil {
						continue
					}
					typeFromField := requestField.Desc.Kind().String()
					if requestField.Desc.IsList() {
						typeFromField = fmt.Sprintf("[]%s", typeFromField)
					}
					switch typeFromField {
					case stringType:
						g.P("if request.", requestField.GoName, " != \"\" {")
						g.P("x.", requestField.GoName, " = request.Get", requestField.GoName, "()")
						g.P("}")
					case uint32Type:
						g.P("if request.", requestField.GoName, " != 0 {")
						g.P("x.", requestField.GoName, " = request.Get", requestField.GoName, "()")
						g.P("}")
					case "[]string", "[]uint32":
						g.P("if len(request.", requestField.GoName, ") > 0 {")
						g.P("x.", requestField.GoName, " = request.Get", requestField.GoName, "()")
						g.P("}")
					default:
						g.P("x.", requestField.GoName, " = request.Get", requestField.GoName, "()")
					}
				}
				g.P("}")
			}
		}
		if strings.Contains(string(msg.Comments.Trailing), "@merge:") {
			re := regexp.MustCompile(`(?m)@merge:"(.*)\|(.*)"`)
			for _, match := range re.FindAllStringSubmatch(string(msg.Comments.Trailing), -1) {
				modelsForMerge := strings.Split(match[1], ",")
				for _, modelForMerge := range modelsForMerge {

					requestFields := getFieldsFromMessage(file.Messages, modelForMerge)
					g.P()

					g.P("func (x *", Pascal(match[2]), ") MergeFrom", modelForMerge, "(request *", modelForMerge, ") {")
					g.P("if x == nil { return }")
					for _, requestField := range requestFields {
						if strings.Contains(requestField.Comments.Leading.String(), "In: body") {
							typeFromField := requestField.Desc.Kind().String()
							if requestField.Desc.IsList() {
								typeFromField = fmt.Sprintf("[]%s", typeFromField)
							}
							switch typeFromField {
							case stringType:
								g.P("if request.", requestField.GoName, " != \"\" {")
								g.P("x.", requestField.GoName, " = request.Get", requestField.GoName, "()")
								g.P("}")
							case uint32Type:
								g.P("if request.", requestField.GoName, " != 0 {")
								g.P("x.", requestField.GoName, " = request.Get", requestField.GoName, "()")
								g.P("}")
							case "[]string", "[]uint32":
								g.P("if len(request.", requestField.GoName, ") > 0 {")
								g.P("x.", requestField.GoName, " = request.Get", requestField.GoName, "()")
								g.P("}")
							default:
								g.P("x.", requestField.GoName, " = request.Get", requestField.GoName, "()")
							}

						}
					}
					g.P("}")
				}
			}
		}
		/*
			g.P()

			g.P("func (x *", msg.GoIdent, ") MustMarshalBinary() []byte {")
			g.P("b, err := ", g.QualifiedGoIdent(protogen.GoIdent{GoName: "Marshal(x)", GoImportPath: "github.com/goccy/go-json"}))
			g.P("if err != nil { ", g.QualifiedGoIdent(fmtPackage.Ident("Println(err)")), " }")
			g.P("return b")
			g.P("}")

			g.P()

			g.P("func (x *", msg.GoIdent, ") MarshalBinary() ([]byte, error) {")
			g.P("return ", g.QualifiedGoIdent(protogen.GoIdent{GoName: "Marshal(x)", GoImportPath: "github.com/goccy/go-json"}))
			g.P("}")

			g.P()

			g.P("func (x *", msg.GoIdent, ") UnmarshalBinary( data []byte) error {")
			g.P("if err := ", g.QualifiedGoIdent(protogen.GoIdent{GoName: "Unmarshal(data, x)", GoImportPath: "github.com/goccy/go-json"}), ";err != nil {")
			g.P("return err")
			g.P("}")
			g.P("return nil")
			g.P("}")*/
	}
	return g
}

const (
	INITIAL_STATE                 = iota
	EXPECT_FOLLOWING_SMALL_LETTER = iota
	IN_CONSECUTIVE_CAPITALS       = iota
	IN_WORD                       = iota
	SEEK_FOR_NEXT_WORD            = iota
)

type CaseTranslator struct {
	FirstLetter       func(rune) rune
	LetterInWord      func(rune) rune
	FirstLetterOfWord func(rune) rune
	Separator         rune
}

type processor struct {
	state        int
	buffer       *bytes.Buffer
	tr           *CaseTranslator
	bufferedRune rune
}

func NewProcessor(t *CaseTranslator) *processor {
	p := new(processor)
	p.state = INITIAL_STATE
	p.buffer = bytes.NewBuffer(nil)
	p.tr = t
	return p
}

func (p *processor) flushRuneBuffer() {
	if p.bufferedRune != 0 {
		p.writeRune(p.bufferedRune)
	}
}
func (p *processor) putCharInRuneBuffer(r rune) {
	if p.bufferedRune != 0 {
		p.charInWord(p.bufferedRune)
	}
	p.bufferedRune = r
}
func (p *processor) writeRune(r rune) {
	p.buffer.WriteRune(r)
}
func (p *processor) firstLetter(r rune) {
	p.writeRune(p.tr.FirstLetter(r))
	if unicode.IsUpper(r) {
		p.state = EXPECT_FOLLOWING_SMALL_LETTER
	} else {
		p.state = IN_WORD
	}
}
func (p *processor) foundNewWord(r rune) {
	if p.tr.Separator != 0 {
		p.writeRune(p.tr.Separator)
	}
	p.writeRune(p.tr.FirstLetterOfWord(r))

	if unicode.IsUpper(r) {
		p.state = EXPECT_FOLLOWING_SMALL_LETTER
	} else {
		p.state = IN_WORD
	}
}

func (p *processor) charInWord(r rune) {
	r = p.tr.LetterInWord(r)
	p.writeRune(r)
}
func (p *processor) firstLetterOfWord(r rune) {
	r = p.tr.FirstLetterOfWord(r)
	p.writeRune(r)
}

func (p *processor) convert(s string) string {
	p.buffer.Grow(len(s))
	for _, r := range s {
		isNumber := unicode.Is(unicode.Number, r)
		isWord := unicode.Is(unicode.Letter, r) || isNumber

		switch p.state {
		case INITIAL_STATE:
			if isWord {
				p.firstLetter(r)
			}
		case EXPECT_FOLLOWING_SMALL_LETTER:
			if isWord {
				if unicode.IsUpper(r) {
					p.putCharInRuneBuffer(r)
					p.state = IN_CONSECUTIVE_CAPITALS
				} else {
					p.flushRuneBuffer()
					p.charInWord(r)
					p.state = IN_WORD
				}
			} else {
				p.putCharInRuneBuffer(0)
				p.state = SEEK_FOR_NEXT_WORD
			}
		case IN_CONSECUTIVE_CAPITALS:
			if isWord {
				if unicode.IsUpper(r) || isNumber {
					p.putCharInRuneBuffer(r)
				} else {
					p.foundNewWord(p.bufferedRune)
					p.bufferedRune = 0
					p.charInWord(r)
					p.state = IN_WORD
				}
			} else {
				p.putCharInRuneBuffer(0)
				p.state = SEEK_FOR_NEXT_WORD
			}
		case IN_WORD:
			if isWord {
				if unicode.IsUpper(r) {
					p.foundNewWord(r)
				} else {
					p.charInWord(r)
				}
			} else {
				p.state = SEEK_FOR_NEXT_WORD
			}
		case SEEK_FOR_NEXT_WORD:
			if isWord {
				p.foundNewWord(r)
			}
		}
	}
	if p.bufferedRune != 0 {
		p.charInWord(p.bufferedRune)
	}
	return p.buffer.String()
}

func (p *processor) Convert(s string) string {
	return p.convert(s)
}

func NewLowerProcessor(separator rune) *processor {
	return NewProcessor(&CaseTranslator{
		FirstLetter:       unicode.ToLower,
		LetterInWord:      unicode.ToLower,
		FirstLetterOfWord: unicode.ToLower,
		Separator:         separator,
	})
}

func Camel(s string) string {
	return NewProcessor(&CaseTranslator{
		FirstLetter:       unicode.ToLower,
		LetterInWord:      unicode.ToLower,
		FirstLetterOfWord: unicode.ToUpper,
	}).Convert(s)
}

func Pascal(s string) string {
	return NewProcessor(&CaseTranslator{
		FirstLetter:       unicode.ToUpper,
		LetterInWord:      unicode.ToLower,
		FirstLetterOfWord: unicode.ToUpper,
	}).Convert(s)
}

func Snake(s string) string {
	return NewLowerProcessor('_').Convert(s)
}
