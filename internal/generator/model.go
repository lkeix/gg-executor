package generator

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/lkeix/gg-executor/schema"
)

func generateModelImport() *ast.GenDecl {
	return &ast.GenDecl{
		Tok: token.IMPORT,
		Specs: []ast.Spec{
			&ast.ImportSpec{
				Path: &ast.BasicLit{
					Kind:  token.STRING,
					Value: `"encoding/json"`,
				},
			},
			&ast.ImportSpec{
				Path: &ast.BasicLit{
					Kind:  token.STRING,
					Value: `"fmt"`,
				},
			},
		},
	}
}

func generateModelField(field schema.FieldDefinitions) *ast.FieldList {
	fields := make([]*ast.Field, 0, len(field))

	for _, f := range field {
		fieldType := GraphQLType(f.Type.Name)
		var fieldTypeIdent *ast.Ident
		if fieldType.IsPrimitive() {
			fieldTypeIdent = golangType(f.Type, fieldType, "")
		} else {
			fieldTypeIdent = ast.NewIdent(string(fieldType))
		}

		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{
				{
					Name: toUpperCase(string(f.Name)),
				},
			},
			Type: fieldTypeIdent,
			Tag: &ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf("`json:\"%s\"`", string(f.Name)),
			},
		})
	}

	return &ast.FieldList{
		List: fields,
	}
}

func generateTypeModelMarshalJSON(t *schema.TypeDefinition) *ast.FuncDecl {
	mappingSchemaValidation := generateMappingSchemaValidation(t)

	stmts := []ast.Stmt{}
	stmts = append(stmts, mappingSchemaValidation...)

	stmts = append(stmts, &ast.ReturnStmt{
		Results: []ast.Expr{
			&ast.CallExpr{
				Args: []ast.Expr{
					&ast.Ident{
						Name: "t",
					},
				},
				Fun: &ast.SelectorExpr{
					X: &ast.Ident{
						Name: "json",
					},
					Sel: &ast.Ident{
						Name: "Marshal",
					},
				},
			},
		},
	})

	return &ast.FuncDecl{
		Name: ast.NewIdent("MarshalJSON"),
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{
						{
							Name: "t",
						},
					},
					Type: &ast.StarExpr{
						X: &ast.Ident{
							Name: string(t.Name),
						},
					},
				},
			},
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: &ast.Ident{
							Name: "[]byte",
						},
					},
					{
						Type: &ast.Ident{
							Name: "error",
						},
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: stmts,
		},
	}
}


func generateModelMapperField(field schema.FieldDefinitions) *ast.FieldList {
	fields := make([]*ast.Field, 0, len(field))

	starExpr := func(fieldType *schema.FieldType, graphQLType GraphQLType, modelPackagePath string) *ast.StarExpr {
		if fieldType.IsList {
			return &ast.StarExpr{
				X: &ast.Ident{
					Name: "[]" + golangType(fieldType.ListType, GraphQLType(fieldType.ListType.Name), modelPackagePath).Name,
				},
			}
		}

		if graphQLType.IsPrimitive() {
			if fieldType.Nullable {
				return &ast.StarExpr{
					X: &ast.Ident{
						Name: graphQLType.golangType(),
					},
				}
			}
		}

		return &ast.StarExpr{
			X: &ast.Ident{
				Name: graphQLType.golangType(),
			},
		}
	}

	for _, f := range field {
		fieldType := GraphQLType(f.Type.Name)
		var fieldTypeIdent *ast.StarExpr
		if fieldType.IsPrimitive() {
			fieldTypeIdent = starExpr(f.Type, fieldType, "")
		} else {
			fieldTypeIdent = starExpr(f.Type, fieldType, "")
		}

		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{
				{
					Name: toUpperCase(string(f.Name)),
				},
			},
			Type: fieldTypeIdent,
			Tag: &ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf("`json:\"%s\"`", string(f.Name)),
			},
		})
	}

	return &ast.FieldList{
		List: fields,
	}
}

func generateInputModelUnmarshalJSON(t *schema.InputDefinition) *ast.FuncDecl {
	var stmts []ast.Stmt
	stmts = append(stmts, generateUnmarshalJSONBody(t.Fields)...)
	stmts = append(stmts, generateMappingSchemaValidation(t)...)
	stmts = append(stmts, generateMapping(t.Fields)...)

	return &ast.FuncDecl{
		Name: ast.NewIdent("UnmarshalJSON"),
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{
						{
							Name: "t",
						},
					},
					Type: &ast.StarExpr{
						X: &ast.Ident{
							Name: string(t.Name),
						},
					},
				},
			},
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{
							{
								Name: "data",
							},
						},
						Type: &ast.Ident{
							Name: "[]byte",
						},
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{
						Type: &ast.Ident{
							Name: "error",
						},
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: append(stmts, &ast.ReturnStmt{
				Results: []ast.Expr{
					ast.NewIdent("nil"),
				},
			}),
		},
	}
}

func generateUnmarshalJSONBody(fields schema.FieldDefinitions) []ast.Stmt {
	return []ast.Stmt{
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{
							{
								Name: "mapper",
							},
						},
						Type: &ast.StructType{
							Fields: &ast.FieldList{
								List: []*ast.Field{
									{
										Names: []*ast.Ident{
											ast.NewIdent("Data"),
										},
										Type: &ast.StructType{
											Fields: generateModelMapperField(fields),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		&ast.IfStmt{
			Init: &ast.AssignStmt{
				Lhs: []ast.Expr{
					ast.NewIdent("err"),
				},
				Rhs: []ast.Expr{
					&ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X: &ast.Ident{
								Name: "json",
							},
							Sel: &ast.Ident{
								Name: "Unmarshal",
							},
						},
						Args: []ast.Expr{
							ast.NewIdent("data"),
							ast.NewIdent("&mapper"),
						},
					},
				},
				Tok: token.DEFINE,
			},
			Cond: &ast.BinaryExpr{
				X:  ast.NewIdent("err"),
				Op: token.NEQ,
				Y:  ast.NewIdent("nil"),
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ReturnStmt{
						Results: []ast.Expr{
							ast.NewIdent("err"),
						},
					},
				},
			},
		},
	}
}

func generateMapping(fields schema.FieldDefinitions) []ast.Stmt {
	stmts := make([]ast.Stmt, 0, len(fields))

	for _, f := range fields {
		var field ast.Expr
		field = &ast.SelectorExpr{
			X: &ast.SelectorExpr{
				X:   ast.NewIdent("mapper"),
				Sel: ast.NewIdent("Data"),
			},
			Sel: ast.NewIdent(toUpperCase(string(f.Name))),
		}
		if !f.Type.Nullable {
			field = &ast.StarExpr{
				X: field,
			}
		}

		stmts = append(stmts, &ast.AssignStmt{
			Lhs: []ast.Expr{
				&ast.SelectorExpr{
					X: &ast.Ident{
						Name: "t",
					},
					Sel: ast.NewIdent(toUpperCase(string(f.Name))),
				},
			},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				field,
			},
		})
	}

	return stmts
}

func generateMappingSchemaValidation[T *schema.InputDefinition | *schema.TypeDefinition](t T) []ast.Stmt {
	var schemaDefinition any = t
	var selectorX ast.Expr
	switch schemaDefinition.(type) {
	case *schema.InputDefinition:
		selectorX = ast.NewIdent("mapper")
	case *schema.TypeDefinition:
		selectorX = ast.NewIdent("t")
	}

	generateInputIfStmts := func(fields schema.FieldDefinitions) []ast.Stmt {
		stmts := make([]ast.Stmt, 0, len(fields))

		for _, f := range fields {
			if !f.Type.Nullable {
				stmts = append(stmts, &ast.IfStmt{
					Cond: &ast.BinaryExpr{
						X: &ast.SelectorExpr{
							X:   selectorX,
							Sel: ast.NewIdent("Data." + toUpperCase(string(f.Name))),
						},
						Op: token.EQL,
						Y:  ast.NewIdent("nil"),
					},
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							&ast.ReturnStmt{
								Results: []ast.Expr{
									&ast.CallExpr{
										Fun: &ast.SelectorExpr{
											X: &ast.Ident{
												Name: "fmt",
											},
											Sel: ast.NewIdent("Errorf"),
										},
										Args: []ast.Expr{
											&ast.BasicLit{
												Kind:  token.STRING,
												Value: fmt.Sprintf("`%s is required`", string(f.Name)),
											},
										},
									},
								},
							},
						},
					},
				})
			}
		}

		return stmts
	}

	generateTypeIfStmts := func(fields schema.FieldDefinitions) []ast.Stmt {
		stmts := make([]ast.Stmt, 0, len(fields))

		for _, f := range fields {
			if !f.Type.Nullable {
				stmts = append(stmts, &ast.IfStmt{
					Cond: &ast.BinaryExpr{
						X: &ast.SelectorExpr{
							X:   selectorX,
							Sel: ast.NewIdent(toUpperCase(string(f.Name))),
						},
						Op: token.EQL,
						Y:  ast.NewIdent("nil"),
					},
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							&ast.ReturnStmt{
								Results: []ast.Expr{
									ast.NewIdent("nil"),
									&ast.CallExpr{
										Fun: &ast.SelectorExpr{
											X: &ast.Ident{
												Name: "fmt",
											},
											Sel: ast.NewIdent("Errorf"),
										},
										Args: []ast.Expr{
											&ast.BasicLit{
												Kind:  token.STRING,
												Value: fmt.Sprintf("`%s is required`", string(f.Name)),
											},
										},
									},
								},
							},
						},
					},
				})
			}
		}

		return stmts
	}

	var definition any = t
	switch d := definition.(type) {
	case *schema.InputDefinition:
		return generateInputIfStmts(d.Fields)
	case *schema.TypeDefinition:
		return generateTypeIfStmts(d.Fields)
	}

	return []ast.Stmt{}
}
