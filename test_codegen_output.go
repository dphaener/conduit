package main

import (
	"fmt"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/compiler/codegen"
)

func main() {
	gen := codegen.NewGenerator()
	r := &ast.ResourceNode{
		Name: "User",
		Fields: []*ast.FieldNode{
			{
				Name: "username",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				Nullable: false,
			},
		},
	}
	code, _ := gen.GenerateResource(r)
	fmt.Println(code)
}
