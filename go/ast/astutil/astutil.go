package astutil

import (
	"go/ast"
)

// Clone returns a deep copy of node.
func Clone(node ast.Node) ast.Node {
	switch node := node.(type) {
	case *ast.ArrayType:
		return &ast.ArrayType{
			Lbrack: node.Lbrack,
			Len:    cloneExpr(node.Len),
			Elt:    cloneExpr(node.Elt),
		}
	case *ast.AssignStmt:
		return &ast.AssignStmt{
			Lhs:    cloneExprSlice(node.Lhs),
			TokPos: node.TokPos,
			Tok:    node.Tok,
			Rhs:    cloneExprSlice(node.Rhs),
		}
	case *ast.BadDecl:
		return &ast.BadDecl{
			From: node.From,
			To:   node.To,
		}
	case *ast.BadExpr:
		return &ast.BadExpr{
			From: node.From,
			To:   node.To,
		}
	case *ast.BadStmt:
		return &ast.BadStmt{
			From: node.From,
			To:   node.To,
		}
	case *ast.BasicLit:
		return &ast.BasicLit{
			ValuePos: node.ValuePos,
			Kind:     node.Kind,
			Value:    node.Value,
		}
	case *ast.BinaryExpr:
		return &ast.BinaryExpr{
			X:     cloneExpr(node.X),
			OpPos: node.OpPos,
			Op:    node.Op,
			Y:     cloneExpr(node.Y),
		}
	case *ast.BlockStmt:
		return &ast.BlockStmt{
			Lbrace: node.Lbrace,
			List:   cloneStmtSlice(node.List),
			Rbrace: node.Rbrace,
		}
	case *ast.BranchStmt:
		return &ast.BranchStmt{
			TokPos: node.TokPos,
			Tok:    node.Tok,
			Label:  node.Label,
		}
	case *ast.CallExpr:
		return &ast.CallExpr{
			Fun:      node.Fun,
			Lparen:   node.Lparen,
			Args:     cloneExprSlice(node.Args),
			Ellipsis: node.Ellipsis,
			Rparen:   node.Rparen,
		}
	case *ast.CaseClause:
		return &ast.CaseClause{
			Case:  node.Case,
			List:  cloneExprSlice(node.List),
			Colon: node.Colon,
			Body:  cloneStmtSlice(node.Body),
		}
	case *ast.ChanType:
		return &ast.ChanType{
			Begin: node.Begin,
			Arrow: node.Arrow,
			Dir:   node.Dir,
			Value: cloneExpr(node.Value),
		}
	case *ast.CommClause:
		return &ast.CommClause{
			Case:  node.Case,
			Comm:  cloneStmt(node.Comm),
			Colon: node.Colon,
			Body:  cloneStmtSlice(node.Body),
		}
	case *ast.Comment:
		return &ast.Comment{
			Slash: node.Slash,
			Text:  node.Text,
		}
	case *ast.CommentGroup:
		return &ast.CommentGroup{
			List: cloneCommentSlice(node.List),
		}
	case *ast.CompositeLit:
		return &ast.CompositeLit{
			Type:       cloneExpr(node.Type),
			Lbrace:     node.Lbrace,
			Elts:       cloneExprSlice(node.Elts),
			Rbrace:     node.Rbrace,
			Incomplete: node.Incomplete,
		}
	case *ast.DeclStmt:
		return &ast.DeclStmt{
			Decl: cloneDecl(node.Decl),
		}
	case *ast.DeferStmt:
		return &ast.DeferStmt{
			Defer: node.Defer,
			Call:  cloneCallExpr(node.Call),
		}
	case *ast.Ellipsis:
		return &ast.Ellipsis{
			Ellipsis: node.Ellipsis,
			Elt:      cloneExpr(node.Elt),
		}
	case *ast.EmptyStmt:
		return &ast.EmptyStmt{
			Semicolon: node.Semicolon,
			Implicit:  node.Implicit,
		}
	case *ast.ExprStmt:
		return &ast.ExprStmt{
			X: cloneExpr(node.X),
		}
	case *ast.Field:
		return &ast.Field{
			Doc:     cloneCommentGroup(node.Doc),
			Names:   cloneIdentSlice(node.Names),
			Type:    cloneExpr(node.Type),
			Tag:     cloneBasicLit(node.Tag),
			Comment: cloneCommentGroup(node.Comment),
		}
	case *ast.FieldList:
		return &ast.FieldList{
			Opening: node.Opening,
			List:    cloneFieldSlice(node.List),
			Closing: node.Closing,
		}
	case *ast.File:
		return &ast.File{
			Doc:        cloneCommentGroup(node.Doc),
			Package:    node.Package,
			Name:       cloneIdent(node.Name),
			Decls:      cloneDeclSlice(node.Decls),
			Scope:      cloneScope(node.Scope),
			Imports:    cloneImportSpecSlice(node.Imports),
			Unresolved: cloneIdentSlice(node.Unresolved),
			Comments:   cloneCommentGroupSlice(node.Comments),
		}
	case *ast.ForStmt:
		return &ast.ForStmt{
			For:  node.For,
			Init: cloneStmt(node.Init),
			Cond: cloneExpr(node.Cond),
			Post: cloneStmt(node.Post),
			Body: cloneBlockStmt(node.Body),
		}
	case *ast.FuncDecl:
		return &ast.FuncDecl{
			Doc:  cloneCommentGroup(node.Doc),
			Recv: cloneFieldList(node.Recv),
			Name: cloneIdent(node.Name),
			Type: cloneFuncType(node.Type),
			Body: cloneBlockStmt(node.Body),
		}
	case *ast.FuncLit:
		return &ast.FuncLit{
			Type: cloneFuncType(node.Type),
			Body: cloneBlockStmt(node.Body),
		}
	case *ast.FuncType:
		return &ast.FuncType{
			Params:  cloneFieldList(node.Params),
			Results: cloneFieldList(node.Results),
		}
	case *ast.GenDecl:
		return &ast.GenDecl{
			Doc:    cloneCommentGroup(node.Doc),
			TokPos: node.TokPos,
			Tok:    node.Tok,
			Lparen: node.Lparen,
			Specs:  cloneSpecSlice(node.Specs),
			Rparen: node.Rparen,
		}
	case *ast.GoStmt:
		return &ast.GoStmt{
			Go:   node.Go,
			Call: cloneCallExpr(node.Call),
		}
	case *ast.Ident:
		return &ast.Ident{
			NamePos: node.NamePos,
			Name:    node.Name,
			Obj:     cloneObject(node.Obj),
		}
	case *ast.IfStmt:
		return &ast.IfStmt{
			If:   node.If,
			Init: cloneStmt(node.Init),
			Cond: cloneExpr(node.Cond),
			Body: cloneBlockStmt(node.Body),
			Else: cloneStmt(node.Else),
		}
	case *ast.ImportSpec:
		return &ast.ImportSpec{
			Doc:     cloneCommentGroup(node.Doc),
			Name:    cloneIdent(node.Name),
			Path:    cloneBasicLit(node.Path),
			Comment: cloneCommentGroup(node.Comment),
			EndPos:  node.EndPos,
		}
	case *ast.IncDecStmt:
		return &ast.IncDecStmt{
			X:      cloneExpr(node.X),
			TokPos: node.TokPos,
			Tok:    node.Tok,
		}
	case *ast.IndexExpr:
		return &ast.IndexExpr{
			X:      cloneExpr(node.X),
			Lbrack: node.Lbrack,
			Index:  cloneExpr(node.Index),
			Rbrack: node.Rbrack,
		}
	case *ast.InterfaceType:
		return &ast.InterfaceType{
			Interface:  node.Interface,
			Methods:    cloneFieldList(node.Methods),
			Incomplete: node.Incomplete,
		}
	case *ast.KeyValueExpr:
		return &ast.KeyValueExpr{
			Key:   cloneExpr(node.Key),
			Colon: node.Colon,
			Value: cloneExpr(node.Value),
		}
	case *ast.LabeledStmt:
		return &ast.LabeledStmt{
			Label: cloneIdent(node.Label),
			Colon: node.Colon,
			Stmt:  cloneStmt(node.Stmt),
		}
	case *ast.MapType:
		return &ast.MapType{
			Map:   node.Map,
			Key:   cloneExpr(node.Key),
			Value: cloneExpr(node.Value),
		}
	case *ast.Package:
		var imports map[string]*ast.Object
		if node.Imports != nil {
			imports = make(map[string]*ast.Object)
		}
		for k, v := range node.Imports {
			imports[k] = cloneObject(v)
		}

		var files map[string]*ast.File
		if node.Files != nil {
			files = make(map[string]*ast.File)
		}
		for k, v := range node.Files {
			files[k] = cloneFile(v)
		}

		return &ast.Package{
			Name:    node.Name,
			Scope:   cloneScope(node.Scope),
			Imports: imports,
			Files:   files,
		}
	case *ast.ParenExpr:
		return &ast.ParenExpr{
			Lparen: node.Lparen,
			X:      cloneExpr(node.X),
			Rparen: node.Rparen,
		}
	case *ast.RangeStmt:
		return &ast.RangeStmt{
			For:    node.For,
			Key:    cloneExpr(node.Key),
			Value:  cloneExpr(node.Value),
			TokPos: node.TokPos,
			Tok:    node.Tok,
			X:      cloneExpr(node.X),
			Body:   cloneBlockStmt(node.Body),
		}
	case *ast.ReturnStmt:
		return &ast.ReturnStmt{
			Return:  node.Return,
			Results: cloneExprSlice(node.Results),
		}
	case *ast.SelectStmt:
		return &ast.SelectStmt{
			Select: node.Select,
			Body:   cloneBlockStmt(node.Body),
		}
	case *ast.SelectorExpr:
		return &ast.SelectorExpr{
			X:   cloneExpr(node.X),
			Sel: cloneIdent(node.Sel),
		}
	case *ast.SendStmt:
		return &ast.SendStmt{
			Chan:  cloneExpr(node.Chan),
			Arrow: node.Arrow,
			Value: cloneExpr(node.Value),
		}
	case *ast.SliceExpr:
		return &ast.SliceExpr{
			X:      cloneExpr(node.X),
			Lbrack: node.Lbrack,
			Low:    cloneExpr(node.Low),
			High:   cloneExpr(node.High),
			Max:    cloneExpr(node.Max),
			Slice3: node.Slice3,
			Rbrack: node.Rbrack,
		}
	case *ast.StarExpr:
		return &ast.StarExpr{
			Star: node.Star,
			X:    cloneExpr(node.X),
		}
	case *ast.StructType:
		return &ast.StructType{
			Struct:     node.Struct,
			Fields:     cloneFieldList(node.Fields),
			Incomplete: node.Incomplete,
		}
	case *ast.SwitchStmt:
		return &ast.SwitchStmt{
			Switch: node.Switch,
			Init:   cloneStmt(node.Init),
			Tag:    cloneExpr(node.Tag),
			Body:   cloneBlockStmt(node.Body),
		}
	case *ast.TypeAssertExpr:
		return &ast.TypeAssertExpr{
			X:      cloneExpr(node.X),
			Lparen: node.Lparen,
			Type:   cloneExpr(node.Type),
			Rparen: node.Rparen,
		}
	case *ast.TypeSpec:
		return &ast.TypeSpec{
			Doc:     cloneCommentGroup(node.Doc),
			Name:    cloneIdent(node.Name),
			Assign:  node.Assign,
			Type:    cloneExpr(node.Type),
			Comment: cloneCommentGroup(node.Comment),
		}
	case *ast.TypeSwitchStmt:
		return &ast.TypeSwitchStmt{
			Switch: node.Switch,
			Init:   cloneStmt(node.Init),
			Assign: cloneStmt(node.Assign),
			Body:   cloneBlockStmt(node.Body),
		}
	case *ast.UnaryExpr:
		return &ast.UnaryExpr{
			OpPos: node.OpPos,
			Op:    node.Op,
			X:     cloneExpr(node.X),
		}
	case *ast.ValueSpec:
		return &ast.ValueSpec{
			Doc:     cloneCommentGroup(node.Doc),
			Names:   cloneIdentSlice(node.Names),
			Type:    cloneExpr(node.Type),
			Values:  cloneExprSlice(node.Values),
			Comment: cloneCommentGroup(node.Comment),
		}
	default:
		panic("unreachable")
	}
}

func cloneExpr(expr ast.Expr) ast.Expr {
	if expr == nil {
		return nil
	}
	return Clone(expr).(ast.Expr)
}

func cloneExprSlice(a []ast.Expr) []ast.Expr {
	if a == nil {
		return nil
	}
	other := make([]ast.Expr, len(a))
	for i := range a {
		other[i] = cloneExpr(a[i])
	}
	return other
}

func cloneStmt(stmt ast.Stmt) ast.Stmt {
	if stmt == nil {
		return nil
	}
	return Clone(stmt).(ast.Stmt)
}

func cloneStmtSlice(a []ast.Stmt) []ast.Stmt {
	if a == nil {
		return nil
	}
	other := make([]ast.Stmt, len(a))
	for i := range a {
		other[i] = cloneStmt(a[i])
	}
	return other
}

func cloneComment(comment *ast.Comment) *ast.Comment {
	if comment == nil {
		return nil
	}
	return Clone(comment).(*ast.Comment)
}

func cloneCommentSlice(a []*ast.Comment) []*ast.Comment {
	if a == nil {
		return nil
	}
	other := make([]*ast.Comment, len(a))
	for i := range a {
		other[i] = cloneComment(a[i])
	}
	return other
}

func cloneDecl(decl ast.Decl) ast.Decl {
	if decl == nil {
		return nil
	}
	return Clone(decl).(ast.Decl)
}

func cloneDeclSlice(a []ast.Decl) []ast.Decl {
	if a == nil {
		return nil
	}
	other := make([]ast.Decl, len(a))
	for i := range a {
		other[i] = cloneDecl(a[i])
	}
	return other
}

func cloneCallExpr(expr *ast.CallExpr) *ast.CallExpr {
	if expr == nil {
		return nil
	}
	return Clone(expr).(*ast.CallExpr)
}

func cloneCommentGroup(group *ast.CommentGroup) *ast.CommentGroup {
	if group == nil {
		return nil
	}
	return Clone(group).(*ast.CommentGroup)
}

func cloneCommentGroupSlice(a []*ast.CommentGroup) []*ast.CommentGroup {
	if a == nil {
		return nil
	}
	other := make([]*ast.CommentGroup, len(a))
	for i := range a {
		other[i] = cloneCommentGroup(a[i])
	}
	return other
}

func cloneIdent(ident *ast.Ident) *ast.Ident {
	if ident == nil {
		return nil
	}
	return Clone(ident).(*ast.Ident)
}

func cloneIdentSlice(a []*ast.Ident) []*ast.Ident {
	if a == nil {
		return nil
	}
	other := make([]*ast.Ident, len(a))
	for i := range a {
		other[i] = cloneIdent(a[i])
	}
	return other
}

func cloneBasicLit(node *ast.BasicLit) *ast.BasicLit {
	if node == nil {
		return nil
	}
	return Clone(node).(*ast.BasicLit)
}

func cloneField(node *ast.Field) *ast.Field {
	if node == nil {
		return nil
	}
	return Clone(node).(*ast.Field)
}

func cloneFieldSlice(a []*ast.Field) []*ast.Field {
	if a == nil {
		return nil
	}
	other := make([]*ast.Field, len(a))
	for i := range a {
		other[i] = cloneField(a[i])
	}
	return other
}

func cloneImportSpec(node *ast.ImportSpec) *ast.ImportSpec {
	if node == nil {
		return nil
	}
	return Clone(node).(*ast.ImportSpec)
}

func cloneImportSpecSlice(a []*ast.ImportSpec) []*ast.ImportSpec {
	if a == nil {
		return nil
	}
	other := make([]*ast.ImportSpec, len(a))
	for i := range a {
		other[i] = cloneImportSpec(a[i])
	}
	return other
}

func cloneBlockStmt(node *ast.BlockStmt) *ast.BlockStmt {
	if node == nil {
		return nil
	}
	return Clone(node).(*ast.BlockStmt)
}

func cloneBlockStmtSlice(a []*ast.BlockStmt) []*ast.BlockStmt {
	if a == nil {
		return nil
	}
	other := make([]*ast.BlockStmt, len(a))
	for i := range a {
		other[i] = cloneBlockStmt(a[i])
	}
	return other
}

func cloneFieldList(node *ast.FieldList) *ast.FieldList {
	if node == nil {
		return nil
	}
	return Clone(node).(*ast.FieldList)
}

func cloneFieldListSlice(a []*ast.FieldList) []*ast.FieldList {
	if a == nil {
		return nil
	}
	other := make([]*ast.FieldList, len(a))
	for i := range a {
		other[i] = cloneFieldList(a[i])
	}
	return other
}

func cloneFuncType(node *ast.FuncType) *ast.FuncType {
	if node == nil {
		return nil
	}
	return Clone(node).(*ast.FuncType)
}

func cloneSpec(node ast.Spec) ast.Spec {
	if node == nil {
		return nil
	}
	return Clone(node).(ast.Spec)
}

func cloneSpecSlice(a []ast.Spec) []ast.Spec {
	if a == nil {
		return nil
	}
	other := make([]ast.Spec, len(a))
	for i := range a {
		other[i] = cloneSpec(a[i])
	}
	return other
}

func cloneObject(obj *ast.Object) *ast.Object {
	if obj == nil {
		return nil
	}
	return &ast.Object{
		Kind: obj.Kind,
		Name: obj.Name,
		Decl: obj.Decl,
		Data: obj.Data,
		Type: obj.Type,
	}
}

func cloneFile(node *ast.File) *ast.File {
	if node == nil {
		return nil
	}
	return Clone(node).(*ast.File)
}

func cloneScope(scope *ast.Scope) *ast.Scope {
	var objects map[string]*ast.Object
	if scope.Objects != nil {
		objects = make(map[string]*ast.Object)
		for k, v := range scope.Objects {
			objects[k] = v
		}
	}
	return &ast.Scope{
		Outer:   cloneScope(scope.Outer),
		Objects: objects,
	}
}
