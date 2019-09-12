package glee_test

import (
	"testing"

	"github.com/benbjohnson/glee"
	"github.com/google/go-cmp/cmp"
)

func TestExprWidth(t *testing.T) {
	t.Run("ConstantExpr", func(t *testing.T) {
		if w := glee.ExprWidth(&glee.ConstantExpr{Value: 0, Width: 8}); w != 8 {
			t.Fatalf("unexpected width: %d", w)
		}
	})
	t.Run("NotOptimizedExpr", func(t *testing.T) {
		if w := glee.ExprWidth(&glee.NotOptimizedExpr{Src: &glee.ConstantExpr{Value: 0, Width: 8}}); w != 8 {
			t.Fatalf("unexpected width: %d", w)
		}
	})
	t.Run("SelectExpr", func(t *testing.T) {
		if w := glee.ExprWidth(&glee.SelectExpr{}); w != 8 {
			t.Fatalf("unexpected width: %d", w)
		}
	})
	t.Run("ConcatExpr", func(t *testing.T) {
		if w := glee.ExprWidth(&glee.ConcatExpr{
			MSB: &glee.ConstantExpr{Value: 0, Width: 8},
			LSB: &glee.ConstantExpr{Value: 0, Width: 16},
		}); w != 24 {
			t.Fatalf("unexpected width: %d", w)
		}
	})
	t.Run("ExtractExpr", func(t *testing.T) {
		if w := glee.ExprWidth(&glee.ExtractExpr{
			Expr:   &glee.ConstantExpr{Value: 0, Width: 32},
			Offset: 8,
			Width:  16,
		}); w != 16 {
			t.Fatalf("unexpected width: %d", w)
		}
	})
	t.Run("NotExpr", func(t *testing.T) {
		if w := glee.ExprWidth(&glee.NotExpr{Expr: &glee.ConstantExpr{Value: 0, Width: 8}}); w != 8 {
			t.Fatalf("unexpected width: %d", w)
		}
	})
	t.Run("CastExpr", func(t *testing.T) {
		if w := glee.ExprWidth(&glee.CastExpr{Src: &glee.ConstantExpr{Value: 0, Width: 8}, Width: 16}); w != 16 {
			t.Fatalf("unexpected width: %d", w)
		}
	})
	t.Run("BinaryExpr", func(t *testing.T) {
		t.Run("Bool", func(t *testing.T) {
			if w := glee.ExprWidth(&glee.BinaryExpr{
				Op:  glee.EQ,
				LHS: &glee.ConstantExpr{Value: 0, Width: 8},
				RHS: &glee.ConstantExpr{Value: 0, Width: 8},
			}); w != 1 {
				t.Fatalf("unexpected width: %d", w)
			}
		})
		t.Run("NonBool", func(t *testing.T) {
			if w := glee.ExprWidth(&glee.BinaryExpr{
				Op:  glee.ADD,
				LHS: &glee.ConstantExpr{Value: 0, Width: 8},
				RHS: &glee.ConstantExpr{Value: 0, Width: 8},
			}); w != 8 {
				t.Fatalf("unexpected width: %d", w)
			}
		})
	})
}

func TestBinaryOp_String(t *testing.T) {
	t.Run("Known", func(t *testing.T) {
		if s := glee.ADD.String(); s != "add" {
			t.Fatalf("unexpected string: %s", s)
		}
	})
	t.Run("Unknown", func(t *testing.T) {
		if s := glee.BinaryOp(100).String(); s != "BinaryOp<100>" {
			t.Fatalf("unexpected string: %s", s)
		}
	})
}

func TestBinaryOp_IsArithmetic(t *testing.T) {
	if !glee.ADD.IsArithmetic() {
		t.Fatal("expected true")
	} else if glee.EQ.IsArithmetic() {
		t.Fatal("expected false")
	}
}

func TestBinaryOp_IsCompare(t *testing.T) {
	if !glee.ULT.IsCompare() {
		t.Fatal("expected true")
	} else if glee.SUB.IsCompare() {
		t.Fatal("expected false")
	}
}

func TestBinaryExpr_String(t *testing.T) {
	expr := &glee.BinaryExpr{Op: glee.ADD, LHS: glee.NewConstantExpr(0, 32), RHS: glee.NewConstantExpr(1, 32)}
	if s := expr.String(); s != "(add (const 0 32) (const 1 32))" {
		t.Fatalf("unexpected string: %s", s)
	}
}

func TestNewBinaryExpr_ADD(t *testing.T) {
	t.Run("Constant", func(t *testing.T) {
		if diff := cmp.Diff(
			glee.NewConstantExpr(10, 8),
			glee.NewBinaryExpr(glee.ADD, glee.NewConstantExpr(6, 8), glee.NewConstantExpr(4, 8)),
		); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("ConstantLHSZero", func(t *testing.T) {
		if diff := cmp.Diff(
			glee.NewConstantExpr(10, 8),
			glee.NewBinaryExpr(glee.ADD, glee.NewConstantExpr(0, 8), glee.NewConstantExpr(10, 8)),
		); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("ConstantBool", func(t *testing.T) {
		if diff := cmp.Diff(
			glee.NewConstantExpr(0, 1),
			glee.NewBinaryExpr(glee.ADD, glee.NewConstantExpr(1, 1), glee.NewConstantExpr(1, 1)),
		); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("SymbolicBool", func(t *testing.T) {
		if diff := cmp.Diff(
			&glee.BinaryExpr{
				Op:  glee.XOR,
				LHS: glee.NewConstantExpr(1, 1),
				RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 1), Width: 1},
			},
			glee.NewBinaryExpr(
				glee.ADD,
				&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 1), Width: 1},
				glee.NewConstantExpr(1, 1),
			),
		); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Associative", func(t *testing.T) {
		t.Run("ConstantLHS", func(t *testing.T) {
			t.Run("ADD", func(t *testing.T) {
				if diff := cmp.Diff(
					&glee.BinaryExpr{
						Op:  glee.ADD,
						LHS: glee.NewConstantExpr(4, 8),
						RHS: glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(1, 32)),
					},
					glee.NewBinaryExpr(
						glee.ADD,
						glee.NewConstantExpr(1, 8),
						&glee.BinaryExpr{Op: glee.ADD, LHS: glee.NewConstantExpr(3, 8), RHS: glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(1, 32))},
					),
				); diff != "" {
					t.Fatal(diff)
				}
			})
			t.Run("SUB", func(t *testing.T) {
				if diff := cmp.Diff(
					&glee.BinaryExpr{
						Op:  glee.SUB,
						LHS: glee.NewConstantExpr(4, 8),
						RHS: glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(1, 32)),
					},
					glee.NewBinaryExpr(
						glee.ADD,
						glee.NewConstantExpr(1, 8),
						&glee.BinaryExpr{Op: glee.SUB, LHS: glee.NewConstantExpr(3, 8), RHS: glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(1, 32))},
					),
				); diff != "" {
					t.Fatal(diff)
				}
			})
		})
		t.Run("BinaryLHS", func(t *testing.T) {
			t.Run("ADD", func(t *testing.T) {
				if diff := cmp.Diff(
					&glee.BinaryExpr{
						Op:  glee.ADD,
						LHS: glee.NewConstantExpr(3, 8),
						RHS: &glee.BinaryExpr{
							Op:  glee.ADD,
							LHS: glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(0, 32)),
							RHS: glee.NewSelectExpr(glee.NewArray(0, 2), glee.NewConstantExpr(0, 32)),
						},
					},
					glee.NewBinaryExpr(
						glee.ADD,
						&glee.BinaryExpr{
							Op:  glee.ADD,
							LHS: glee.NewConstantExpr(3, 8),
							RHS: glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(0, 32)),
						},
						glee.NewSelectExpr(glee.NewArray(0, 2), glee.NewConstantExpr(0, 32)),
					),
				); diff != "" {
					t.Fatal(diff)
				}
			})
			t.Run("SUB", func(t *testing.T) {
				if diff := cmp.Diff(
					&glee.BinaryExpr{
						Op:  glee.ADD,
						LHS: glee.NewConstantExpr(3, 8),
						RHS: &glee.BinaryExpr{
							Op:  glee.SUB,
							LHS: glee.NewSelectExpr(glee.NewArray(0, 2), glee.NewConstantExpr(0, 32)),
							RHS: glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(0, 32)),
						},
					},
					glee.NewBinaryExpr(
						glee.ADD,
						&glee.BinaryExpr{
							Op:  glee.SUB,
							LHS: glee.NewConstantExpr(3, 8),
							RHS: glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(0, 32)),
						},
						glee.NewSelectExpr(glee.NewArray(0, 2), glee.NewConstantExpr(0, 32)),
					),
				); diff != "" {
					t.Fatal(diff)
				}
			})
		})
		t.Run("BinaryRHS", func(t *testing.T) {
			t.Run("ADD", func(t *testing.T) {
				if diff := cmp.Diff(
					&glee.BinaryExpr{
						Op:  glee.ADD,
						LHS: glee.NewConstantExpr(3, 8),
						RHS: &glee.BinaryExpr{
							Op:  glee.ADD,
							LHS: glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(0, 32)),
							RHS: glee.NewSelectExpr(glee.NewArray(0, 2), glee.NewConstantExpr(0, 32)),
						},
					},
					glee.NewBinaryExpr(
						glee.ADD,
						glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(0, 32)),
						&glee.BinaryExpr{
							Op:  glee.ADD,
							LHS: glee.NewConstantExpr(3, 8),
							RHS: glee.NewSelectExpr(glee.NewArray(0, 2), glee.NewConstantExpr(0, 32)),
						},
					),
				); diff != "" {
					t.Fatal(diff)
				}
			})
			t.Run("SUB", func(t *testing.T) {
				if diff := cmp.Diff(
					&glee.BinaryExpr{
						Op:  glee.ADD,
						LHS: glee.NewConstantExpr(3, 8),
						RHS: &glee.BinaryExpr{
							Op:  glee.SUB,
							LHS: glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(0, 32)),
							RHS: glee.NewSelectExpr(glee.NewArray(0, 2), glee.NewConstantExpr(0, 32)),
						},
					},
					glee.NewBinaryExpr(
						glee.ADD,
						glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(0, 32)),
						&glee.BinaryExpr{
							Op:  glee.SUB,
							LHS: glee.NewConstantExpr(3, 8),
							RHS: glee.NewSelectExpr(glee.NewArray(0, 2), glee.NewConstantExpr(0, 32)),
						},
					),
				); diff != "" {
					t.Fatal(diff)
				}
			})
		})
	})
}

func TestNewBinaryExpr_SUB(t *testing.T) {
	t.Run("Constant", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.SUB, glee.NewConstantExpr(6, 8), glee.NewConstantExpr(4, 8))
		exp := glee.NewConstantExpr(2, 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("EqualExprs", func(t *testing.T) {
		a := glee.NewArray(0, 2)
		got := glee.NewBinaryExpr(
			glee.SUB,
			glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)),
			glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)),
		)
		exp := glee.NewConstantExpr(0, 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("ConstantBool", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.SUB, glee.NewConstantExpr(1, 1), glee.NewConstantExpr(1, 1))
		exp := glee.NewConstantExpr(0, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("SymbolicBool", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.SUB,
			glee.NewNotOptimizedExpr(glee.NewConstantExpr(1, 1)),
			glee.NewNotOptimizedExpr(glee.NewConstantExpr(0, 1)),
		)
		exp := &glee.BinaryExpr{
			Op:  glee.XOR,
			LHS: glee.NewNotOptimizedExpr(glee.NewConstantExpr(1, 1)),
			RHS: glee.NewNotOptimizedExpr(glee.NewConstantExpr(0, 1)),
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Associative", func(t *testing.T) {
		t.Run("ConstantLHS", func(t *testing.T) {
			t.Run("ADD", func(t *testing.T) {
				got := glee.NewBinaryExpr(
					glee.SUB,
					glee.NewConstantExpr(5, 8),
					&glee.BinaryExpr{Op: glee.ADD, LHS: glee.NewConstantExpr(3, 8), RHS: glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(1, 32))},
				)
				exp := &glee.BinaryExpr{
					Op:  glee.SUB,
					LHS: glee.NewConstantExpr(2, 8),
					RHS: glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(1, 32)),
				}
				if diff := cmp.Diff(got, exp); diff != "" {
					t.Fatal(diff)
				}
			})
			t.Run("SUB", func(t *testing.T) {
				got := glee.NewBinaryExpr(
					glee.SUB,
					glee.NewConstantExpr(5, 8),
					&glee.BinaryExpr{Op: glee.SUB, LHS: glee.NewConstantExpr(3, 8), RHS: glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(1, 32))},
				)
				exp := &glee.BinaryExpr{
					Op:  glee.ADD,
					LHS: glee.NewConstantExpr(2, 8),
					RHS: glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(1, 32)),
				}
				if diff := cmp.Diff(got, exp); diff != "" {
					t.Fatal(diff)
				}
			})
		})
		t.Run("BinaryLHS", func(t *testing.T) {
			t.Run("ADD", func(t *testing.T) {
				got := glee.NewBinaryExpr(
					glee.SUB,
					&glee.BinaryExpr{
						Op:  glee.ADD,
						LHS: glee.NewConstantExpr(3, 8),
						RHS: glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(0, 32)),
					},
					glee.NewSelectExpr(glee.NewArray(0, 2), glee.NewConstantExpr(0, 32)),
				)
				exp := &glee.BinaryExpr{
					Op:  glee.ADD,
					LHS: glee.NewConstantExpr(3, 8),
					RHS: &glee.BinaryExpr{
						Op:  glee.SUB,
						LHS: glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(0, 32)),
						RHS: glee.NewSelectExpr(glee.NewArray(0, 2), glee.NewConstantExpr(0, 32)),
					},
				}
				if diff := cmp.Diff(got, exp); diff != "" {
					t.Fatal(diff)
				}
			})
			t.Run("SUB", func(t *testing.T) {
				got := glee.NewBinaryExpr(
					glee.SUB,
					&glee.BinaryExpr{
						Op:  glee.SUB,
						LHS: glee.NewConstantExpr(3, 8),
						RHS: glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(0, 32)),
					},
					glee.NewSelectExpr(glee.NewArray(0, 2), glee.NewConstantExpr(0, 32)),
				)
				exp := &glee.BinaryExpr{
					Op:  glee.SUB,
					LHS: glee.NewConstantExpr(3, 8),
					RHS: &glee.BinaryExpr{
						Op:  glee.ADD,
						LHS: glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(0, 32)),
						RHS: glee.NewSelectExpr(glee.NewArray(0, 2), glee.NewConstantExpr(0, 32)),
					},
				}
				if diff := cmp.Diff(got, exp); diff != "" {
					t.Fatal(diff)
				}
			})
		})
		t.Run("BinaryRHS", func(t *testing.T) {
			t.Run("ADD", func(t *testing.T) {
				got := glee.NewBinaryExpr(
					glee.SUB,
					glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(0, 32)),
					&glee.BinaryExpr{
						Op:  glee.ADD,
						LHS: glee.NewConstantExpr(3, 8),
						RHS: glee.NewSelectExpr(glee.NewArray(0, 2), glee.NewConstantExpr(1, 32)),
					},
				)
				exp := &glee.BinaryExpr{
					Op:  glee.ADD,
					LHS: glee.NewConstantExpr(253, 8),
					RHS: &glee.BinaryExpr{
						Op:  glee.SUB,
						LHS: glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(0, 32)),
						RHS: glee.NewSelectExpr(glee.NewArray(0, 2), glee.NewConstantExpr(1, 32)),
					},
				}
				if diff := cmp.Diff(got, exp); diff != "" {
					t.Fatal(diff)
				}
			})
			t.Run("SUB", func(t *testing.T) {
				got := glee.NewBinaryExpr(
					glee.SUB,
					glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(0, 32)),
					&glee.BinaryExpr{
						Op:  glee.SUB,
						LHS: glee.NewConstantExpr(3, 8),
						RHS: glee.NewSelectExpr(glee.NewArray(0, 2), glee.NewConstantExpr(0, 32)),
					},
				)
				exp := &glee.BinaryExpr{
					Op:  glee.ADD,
					LHS: glee.NewConstantExpr(253, 8),
					RHS: &glee.BinaryExpr{
						Op:  glee.ADD,
						LHS: glee.NewSelectExpr(glee.NewArray(0, 1), glee.NewConstantExpr(0, 32)),
						RHS: glee.NewSelectExpr(glee.NewArray(0, 2), glee.NewConstantExpr(0, 32)),
					},
				}
				if diff := cmp.Diff(got, exp); diff != "" {
					t.Fatal(diff)
				}
			})
		})
	})
}

func TestNewBinaryExpr_MUL(t *testing.T) {
	t.Run("Constant", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.MUL, glee.NewConstantExpr(6, 8), glee.NewConstantExpr(4, 8))
		exp := glee.NewConstantExpr(24, 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Bool", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.MUL,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 32), Width: 1},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 32), Width: 1},
		)
		exp := &glee.BinaryExpr{
			Op:  glee.AND,
			LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 32), Width: 1},
			RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 32), Width: 1},
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("ConstantOne", func(t *testing.T) {
		a := glee.NewArray(0, 2)
		got := glee.NewBinaryExpr(glee.MUL, glee.NewConstantExpr(1, 8), glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)))
		exp := glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32))
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("ConstantZero", func(t *testing.T) {
		a := glee.NewArray(0, 2)
		got := glee.NewBinaryExpr(glee.MUL, glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)), glee.NewConstantExpr(0, 8))
		exp := glee.NewConstantExpr(0, 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Symbolic", func(t *testing.T) {
		a := glee.NewArray(0, 2)
		got := glee.NewBinaryExpr(
			glee.MUL,
			glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)),
			glee.NewSelectExpr(a, glee.NewConstantExpr(1, 32)),
		)
		exp := &glee.BinaryExpr{
			Op:  glee.MUL,
			LHS: glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)),
			RHS: glee.NewSelectExpr(a, glee.NewConstantExpr(1, 32)),
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestNewBinaryExpr_DIV(t *testing.T) {
	t.Run("UDIV", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.UDIV, glee.NewConstantExpr(20, 8), glee.NewConstantExpr(7, 8))
		exp := glee.NewConstantExpr(uint64(uint8(20)/uint8(7)), 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("SDIV", func(t *testing.T) {
		tmp := int8(-20)
		got := glee.NewBinaryExpr(glee.SDIV, glee.NewConstantExpr(256-20, 8), glee.NewConstantExpr(7, 8))
		exp := glee.NewConstantExpr(uint64(tmp/int8(7)), 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Bool", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.UDIV, glee.NewConstantExpr(1, 1), &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 32), Width: 1})
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Symbolic", func(t *testing.T) {
		a := glee.NewArray(0, 2)
		got := glee.NewBinaryExpr(
			glee.UDIV,
			glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)),
			glee.NewSelectExpr(a, glee.NewConstantExpr(1, 32)),
		)
		exp := &glee.BinaryExpr{
			Op:  glee.UDIV,
			LHS: glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)),
			RHS: glee.NewSelectExpr(a, glee.NewConstantExpr(1, 32)),
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestNewBinaryExpr_REM(t *testing.T) {
	t.Run("UREM", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.UREM, glee.NewConstantExpr(20, 8), glee.NewConstantExpr(7, 8))
		exp := glee.NewConstantExpr(uint64(uint8(20)%uint8(7)), 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("SREM", func(t *testing.T) {
		tmp := int8(-20)
		got := glee.NewBinaryExpr(glee.SREM, glee.NewConstantExpr(256-20, 8), glee.NewConstantExpr(7, 8))
		exp := glee.NewConstantExpr(uint64(tmp%int8(7)), 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Bool", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.UREM, glee.NewConstantExpr(1, 1), &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 32), Width: 1})
		exp := glee.NewConstantExpr(0, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Symbolic", func(t *testing.T) {
		a := glee.NewArray(0, 2)
		got := glee.NewBinaryExpr(
			glee.UREM,
			glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)),
			glee.NewSelectExpr(a, glee.NewConstantExpr(1, 32)),
		)
		exp := &glee.BinaryExpr{
			Op:  glee.UREM,
			LHS: glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)),
			RHS: glee.NewSelectExpr(a, glee.NewConstantExpr(1, 32)),
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestNewBinaryExpr_AND(t *testing.T) {
	t.Run("Constant", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.AND, glee.NewConstantExpr(0x0F, 8), glee.NewConstantExpr(0xFF, 8))
		exp := glee.NewConstantExpr(0x0F, 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("AllOnes", func(t *testing.T) {
		a := glee.NewArray(0, 2)
		got := glee.NewBinaryExpr(glee.AND, glee.NewConstantExpr(0xFF, 8), glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)))
		exp := glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32))
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Zero", func(t *testing.T) {
		a := glee.NewArray(0, 2)
		got := glee.NewBinaryExpr(glee.AND, glee.NewConstantExpr(0, 8), glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)))
		exp := glee.NewConstantExpr(0, 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Symbolic", func(t *testing.T) {
		a := glee.NewArray(0, 2)
		got := glee.NewBinaryExpr(
			glee.AND,
			glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)),
			glee.NewSelectExpr(a, glee.NewConstantExpr(1, 32)),
		)
		exp := &glee.BinaryExpr{
			Op:  glee.AND,
			LHS: glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)),
			RHS: glee.NewSelectExpr(a, glee.NewConstantExpr(1, 32)),
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestNewBinaryExpr_OR(t *testing.T) {
	t.Run("Constant", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.OR, glee.NewConstantExpr(0x0F, 8), glee.NewConstantExpr(0xF8, 8))
		exp := glee.NewConstantExpr(0xFF, 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("AllOnes", func(t *testing.T) {
		a := glee.NewArray(0, 2)
		got := glee.NewBinaryExpr(glee.OR, glee.NewConstantExpr(0xFF, 8), glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)))
		exp := glee.NewConstantExpr(0xFF, 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Zero", func(t *testing.T) {
		a := glee.NewArray(0, 2)
		got := glee.NewBinaryExpr(glee.OR, glee.NewConstantExpr(0, 8), glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)))
		exp := glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32))
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Symbolic", func(t *testing.T) {
		a := glee.NewArray(0, 2)
		got := glee.NewBinaryExpr(
			glee.OR,
			glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)),
			glee.NewSelectExpr(a, glee.NewConstantExpr(1, 32)),
		)
		exp := &glee.BinaryExpr{
			Op:  glee.OR,
			LHS: glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)),
			RHS: glee.NewSelectExpr(a, glee.NewConstantExpr(1, 32)),
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestNewBinaryExpr_XOR(t *testing.T) {
	t.Run("Constant", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.XOR, glee.NewConstantExpr(0x8F, 8), glee.NewConstantExpr(0xF8, 8))
		exp := glee.NewConstantExpr(0x77, 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Zero", func(t *testing.T) {
		a := glee.NewArray(0, 2)
		got := glee.NewBinaryExpr(glee.XOR, glee.NewConstantExpr(0, 8), glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)))
		exp := glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32))
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Bool", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.XOR,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 1), Width: 1},
			glee.NewConstantExpr(0, 1),
		)
		exp := &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 1), Width: 1}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Symbolic", func(t *testing.T) {
		a := glee.NewArray(0, 2)
		got := glee.NewBinaryExpr(
			glee.XOR,
			glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)),
			glee.NewSelectExpr(a, glee.NewConstantExpr(1, 32)),
		)
		exp := &glee.BinaryExpr{
			Op:  glee.XOR,
			LHS: glee.NewSelectExpr(a, glee.NewConstantExpr(0, 32)),
			RHS: glee.NewSelectExpr(a, glee.NewConstantExpr(1, 32)),
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestNewBinaryExpr_SHL(t *testing.T) {
	t.Run("Constant", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.SHL, glee.NewConstantExpr(0x03, 8), glee.NewConstantExpr(4, 8))
		exp := glee.NewConstantExpr(0x30, 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("ConstantBoolShift", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.SHL,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 1), Width: 1},
			glee.NewConstantExpr(3, 8),
		)
		exp := glee.NewConstantExpr(0, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("SymbolicBoolShift", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.SHL,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 1), Width: 1},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
		)
		exp := &glee.BinaryExpr{
			Op:  glee.AND,
			LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 1), Width: 1},
			RHS: &glee.BinaryExpr{
				Op:  glee.EQ,
				LHS: glee.NewConstantExpr(0, 8),
				RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			},
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Symbolic", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.SHL,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
		)
		exp := &glee.BinaryExpr{
			Op:  glee.SHL,
			LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestNewBinaryExpr_LSHR(t *testing.T) {
	t.Run("Constant", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.LSHR, glee.NewConstantExpr(0xF0, 8), glee.NewConstantExpr(4, 8))
		exp := glee.NewConstantExpr(0x0F, 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("ConstantBoolShift", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.LSHR,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 1), Width: 1},
			glee.NewConstantExpr(3, 8),
		)
		exp := glee.NewConstantExpr(0, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("SymbolicBoolShift", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.LSHR,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 1), Width: 1},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
		)
		exp := &glee.BinaryExpr{
			Op:  glee.AND,
			LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 1), Width: 1},
			RHS: &glee.BinaryExpr{
				Op:  glee.EQ,
				LHS: glee.NewConstantExpr(0, 8),
				RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			},
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Symbolic", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.LSHR,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
		)
		exp := &glee.BinaryExpr{
			Op:  glee.LSHR,
			LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestNewBinaryExpr_ASHR(t *testing.T) {
	t.Run("Constant", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.ASHR, glee.NewConstantExpr(0xF0, 8), glee.NewConstantExpr(2, 8))
		exp := glee.NewConstantExpr(0xFC, 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("BoolShift", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.ASHR,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 1), Width: 1},
			glee.NewConstantExpr(3, 8),
		)
		exp := &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 1), Width: 1}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Symbolic", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.ASHR,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
		)
		exp := &glee.BinaryExpr{
			Op:  glee.ASHR,
			LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestNewBinaryExpr_EQ(t *testing.T) {
	t.Run("ConstantTrue", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.EQ, glee.NewConstantExpr(10, 8), glee.NewConstantExpr(10, 8))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("ConstantFalse", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.EQ, glee.NewConstantExpr(3, 8), glee.NewConstantExpr(10, 8))
		exp := glee.NewConstantExpr(0, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Symbolic", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.EQ,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
		)
		exp := &glee.BinaryExpr{
			Op:  glee.EQ,
			LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("SymbolicEqual", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.EQ,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
		)
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("ConstantLHS", func(t *testing.T) {
		t.Run("BinaryExprRHS", func(t *testing.T) {
			t.Run("EQ", func(t *testing.T) {
				t.Run("LHSTrue", func(t *testing.T) {
					got := glee.NewBinaryExpr(
						glee.EQ,
						glee.NewConstantExpr(1, 1),
						&glee.BinaryExpr{
							Op:  glee.EQ,
							LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
							RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
						},
					)
					exp := &glee.BinaryExpr{
						Op:  glee.EQ,
						LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
						RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
					}
					if diff := cmp.Diff(got, exp); diff != "" {
						t.Fatal(diff)
					}
				})
				t.Run("DoubleConstantFalse", func(t *testing.T) {
					got := glee.NewBinaryExpr(
						glee.EQ,
						glee.NewConstantExpr(0, 1),
						&glee.BinaryExpr{
							Op:  glee.EQ,
							LHS: glee.NewConstantExpr(0, 1),
							RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
						},
					)
					exp := &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8}
					if diff := cmp.Diff(got, exp); diff != "" {
						t.Fatal(diff)
					}
				})
			})
			t.Run("OR", func(t *testing.T) {
				t.Run("LHSTrue", func(t *testing.T) {
					got := glee.NewBinaryExpr(
						glee.EQ,
						glee.NewConstantExpr(1, 1),
						&glee.BinaryExpr{
							Op:  glee.OR,
							LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
							RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
						},
					)
					exp := &glee.BinaryExpr{
						Op:  glee.OR,
						LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
						RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
					}
					if diff := cmp.Diff(got, exp); diff != "" {
						t.Fatal(diff)
					}
				})
				t.Run("LHSFalse", func(t *testing.T) {
					got := glee.NewBinaryExpr(
						glee.EQ,
						glee.NewConstantExpr(0, 1),
						&glee.BinaryExpr{
							Op:  glee.OR,
							LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 1},
							RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 1},
						},
					)
					exp := &glee.BinaryExpr{
						Op: glee.AND,
						LHS: &glee.BinaryExpr{
							Op:  glee.EQ,
							LHS: glee.NewConstantExpr(0, 1),
							RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 1},
						},
						RHS: &glee.BinaryExpr{
							Op:  glee.EQ,
							LHS: glee.NewConstantExpr(0, 1),
							RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 1},
						},
					}
					if diff := cmp.Diff(got, exp); diff != "" {
						t.Fatal(diff)
					}
				})
			})
			t.Run("ADD", func(t *testing.T) {
				got := glee.NewBinaryExpr(
					glee.EQ,
					glee.NewConstantExpr(10, 8),
					&glee.BinaryExpr{
						Op:  glee.ADD,
						LHS: glee.NewConstantExpr(3, 8),
						RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
					},
				)
				exp := &glee.BinaryExpr{
					Op:  glee.EQ,
					LHS: glee.NewConstantExpr(7, 8),
					RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
				}
				if diff := cmp.Diff(got, exp); diff != "" {
					t.Fatal(diff)
				}
			})
			t.Run("SUB", func(t *testing.T) {
				got := glee.NewBinaryExpr(
					glee.EQ,
					glee.NewConstantExpr(3, 8),
					&glee.BinaryExpr{
						Op:  glee.SUB,
						LHS: glee.NewConstantExpr(10, 8),
						RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
					},
				)
				exp := &glee.BinaryExpr{
					Op:  glee.EQ,
					LHS: glee.NewConstantExpr(7, 8),
					RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
				}
				if diff := cmp.Diff(got, exp); diff != "" {
					t.Fatal(diff)
				}
			})
		})
		t.Run("CastExprRHS", func(t *testing.T) {
			t.Run("Signed", func(t *testing.T) {
				t.Run("Symbolic", func(t *testing.T) {
					got := glee.NewBinaryExpr(
						glee.EQ,
						glee.NewConstantExpr(1, 16),
						&glee.CastExpr{
							Src:    &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
							Width:  16,
							Signed: true,
						},
					)
					exp := &glee.BinaryExpr{
						Op:  glee.EQ,
						LHS: glee.NewConstantExpr(1, 8),
						RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
					}
					if diff := cmp.Diff(got, exp); diff != "" {
						t.Fatal(diff)
					}
				})
				t.Run("Truncated", func(t *testing.T) {
					got := glee.NewBinaryExpr(
						glee.EQ,
						glee.NewConstantExpr(0x8000, 16),
						&glee.CastExpr{
							Src:    &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
							Width:  16,
							Signed: true,
						},
					)
					exp := glee.NewConstantExpr(0, 1)
					if diff := cmp.Diff(got, exp); diff != "" {
						t.Fatal(diff)
					}
				})
			})
			t.Run("Unsigned", func(t *testing.T) {
				t.Run("Symbolic", func(t *testing.T) {
					got := glee.NewBinaryExpr(
						glee.EQ,
						glee.NewConstantExpr(1, 16),
						&glee.CastExpr{
							Src:   &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
							Width: 16,
						},
					)
					exp := &glee.BinaryExpr{
						Op:  glee.EQ,
						LHS: glee.NewConstantExpr(1, 8),
						RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
					}
					if diff := cmp.Diff(got, exp); diff != "" {
						t.Fatal(diff)
					}
				})
				t.Run("Truncated", func(t *testing.T) {
					got := glee.NewBinaryExpr(
						glee.EQ,
						glee.NewConstantExpr(0x8000, 16),
						&glee.CastExpr{
							Src:   &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
							Width: 16,
						},
					)
					exp := glee.NewConstantExpr(0, 1)
					if diff := cmp.Diff(got, exp); diff != "" {
						t.Fatal(diff)
					}
				})
			})
		})
	})
}

func TestNewBinaryExpr_NE(t *testing.T) {
	t.Run("True", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.NE, glee.NewConstantExpr(1, 8), glee.NewConstantExpr(10, 8))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("False", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.NE, glee.NewConstantExpr(10, 8), glee.NewConstantExpr(10, 8))
		exp := glee.NewConstantExpr(0, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestNewBinaryExpr_ULT(t *testing.T) {
	t.Run("Constant", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.ULT, glee.NewConstantExpr(1, 8), glee.NewConstantExpr(10, 8))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Bool", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.ULT,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 1},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 1},
		)
		exp := &glee.BinaryExpr{
			Op: glee.AND,
			LHS: &glee.BinaryExpr{
				Op:  glee.EQ,
				LHS: glee.NewConstantExpr(0, 1),
				RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 1},
			},
			RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 1},
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Symbolic", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.ULT,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
		)
		exp := &glee.BinaryExpr{
			Op:  glee.ULT,
			LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestNewBinaryExpr_UGT(t *testing.T) {
	t.Run("Constant", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.UGT, glee.NewConstantExpr(1, 8), glee.NewConstantExpr(10, 8))
		exp := glee.NewConstantExpr(0, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Symbolic", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.UGT,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
		)
		exp := &glee.BinaryExpr{
			Op:  glee.ULT,
			LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
			RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestNewBinaryExpr_ULE(t *testing.T) {
	t.Run("Constant", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.ULE, glee.NewConstantExpr(10, 8), glee.NewConstantExpr(10, 8))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Bool", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.ULE,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 1},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 1},
		)
		exp := &glee.BinaryExpr{
			Op: glee.OR,
			LHS: &glee.BinaryExpr{
				Op:  glee.EQ,
				LHS: glee.NewConstantExpr(0, 1),
				RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 1},
			},
			RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 1},
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Symbolic", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.ULE,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
		)
		exp := &glee.BinaryExpr{
			Op:  glee.ULE,
			LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestNewBinaryExpr_UGE(t *testing.T) {
	t.Run("Constant", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.UGE, glee.NewConstantExpr(10, 8), glee.NewConstantExpr(10, 8))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Symbolic", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.UGE,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
		)
		exp := &glee.BinaryExpr{
			Op:  glee.ULE,
			LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
			RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestNewBinaryExpr_SLT(t *testing.T) {
	t.Run("Constant", func(t *testing.T) {
		x := int8(-20)
		got := glee.NewBinaryExpr(glee.SLT, glee.NewConstantExpr(uint64(x), 8), glee.NewConstantExpr(10, 8))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Bool", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.SLT,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 1},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 1},
		)
		exp := &glee.BinaryExpr{
			Op:  glee.AND,
			LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 1},
			RHS: &glee.BinaryExpr{
				Op:  glee.EQ,
				LHS: glee.NewConstantExpr(0, 1),
				RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 1},
			},
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Symbolic", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.SLT,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
		)
		exp := &glee.BinaryExpr{
			Op:  glee.SLT,
			LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestNewBinaryExpr_SGT(t *testing.T) {
	t.Run("Constant", func(t *testing.T) {
		x := int8(-20)
		got := glee.NewBinaryExpr(glee.SGT, glee.NewConstantExpr(uint64(x), 8), glee.NewConstantExpr(10, 8))
		exp := glee.NewConstantExpr(0, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Symbolic", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.SGT,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
		)
		exp := &glee.BinaryExpr{
			Op:  glee.SLT,
			LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
			RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestNewBinaryExpr_SLE(t *testing.T) {
	t.Run("Constant", func(t *testing.T) {
		x := int8(-20)
		got := glee.NewBinaryExpr(glee.SLE, glee.NewConstantExpr(uint64(x), 8), glee.NewConstantExpr(uint64(x), 8))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Bool", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.SLE,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 1},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 1},
		)
		exp := &glee.BinaryExpr{
			Op:  glee.OR,
			LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 1},
			RHS: &glee.BinaryExpr{
				Op:  glee.EQ,
				LHS: glee.NewConstantExpr(0, 1),
				RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 1},
			},
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Symbolic", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.SLE,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
		)
		exp := &glee.BinaryExpr{
			Op:  glee.SLE,
			LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestNewBinaryExpr_SGE(t *testing.T) {
	t.Run("Constant", func(t *testing.T) {
		got := glee.NewBinaryExpr(glee.SGE, glee.NewConstantExpr(10, 8), glee.NewConstantExpr(10, 8))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Symbolic", func(t *testing.T) {
		got := glee.NewBinaryExpr(
			glee.SGE,
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
		)
		exp := &glee.BinaryExpr{
			Op:  glee.SLE,
			LHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(1, 8), Width: 8},
			RHS: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Width: 8},
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestSelectExpr_String(t *testing.T) {
	a := glee.NewArray(0, 2)
	if s := glee.NewSelectExpr(a, glee.NewConstantExpr(0, 8)).String(); s != "(select (array 2) (const 0 8))" {
		t.Fatalf("unexpected string: %s", s)
	}
}

func TestNewConcatExpr(t *testing.T) {
	t.Run("Constant", func(t *testing.T) {
		got := glee.NewConcatExpr(glee.NewConstantExpr(0x80, 8), glee.NewConstantExpr(0xFF, 8))
		exp := glee.NewConstantExpr(0x80FF, 16)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Extract", func(t *testing.T) {
		src := &glee.ExtractExpr{Expr: glee.NewConstantExpr(0x80FF, 16), Width: 16}
		got := glee.NewConcatExpr(
			&glee.ExtractExpr{Expr: src, Offset: 8, Width: 8},
			&glee.ExtractExpr{Expr: src, Offset: 0, Width: 8},
		)
		exp := src
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Symbolic", func(t *testing.T) {
		got := glee.NewConcatExpr(
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Offset: 0, Width: 8},
			&glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Offset: 0, Width: 8},
		)
		exp := &glee.ConcatExpr{
			MSB: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Offset: 0, Width: 8},
			LSB: &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 8), Offset: 0, Width: 8},
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestConcatExpr_String(t *testing.T) {
	expr := &glee.ConcatExpr{MSB: glee.NewConstantExpr(0, 8), LSB: glee.NewConstantExpr(1, 8)}
	if s := expr.String(); s != "(concat (const 0 8) (const 1 8))" {
		t.Fatalf("unexpected string: %s", s)
	}
}

func TestNewExtractExpr(t *testing.T) {
	t.Run("SameWidth", func(t *testing.T) {
		got := glee.NewExtractExpr(glee.NewConstantExpr(100, 16), 0, 16)
		exp := glee.NewConstantExpr(100, 16)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Constant", func(t *testing.T) {
		got := glee.NewExtractExpr(glee.NewConstantExpr(0xFF80, 16), 8, 8)
		exp := glee.NewConstantExpr(0xFF, 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Concat", func(t *testing.T) {
		t.Run("LSBOnly", func(t *testing.T) {
			got := glee.NewExtractExpr(&glee.ConcatExpr{
				MSB: glee.NewConstantExpr(0xDDCC, 16),
				LSB: glee.NewConstantExpr(0xBBAA, 16),
			}, 8, 8)
			exp := glee.NewConstantExpr(0xBB, 8)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
		t.Run("MSBOnly", func(t *testing.T) {
			got := glee.NewExtractExpr(&glee.ConcatExpr{
				MSB: glee.NewConstantExpr(0xDDCC, 16),
				LSB: glee.NewConstantExpr(0xBBAA, 16),
			}, 24, 8)
			exp := glee.NewConstantExpr(0xDD, 8)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
		t.Run("Constant", func(t *testing.T) {
			got := glee.NewExtractExpr(&glee.ConcatExpr{
				MSB: glee.NewConstantExpr(0xDDCC, 16),
				LSB: glee.NewConstantExpr(0xBBAA, 16),
			}, 8, 16)
			exp := glee.NewConstantExpr(0xCCBB, 16)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
		t.Run("Symbolic", func(t *testing.T) {
			got := glee.NewExtractExpr(&glee.ConcatExpr{
				MSB: glee.NewNotOptimizedExpr(glee.NewConstantExpr(0xDDCC, 16)),
				LSB: glee.NewNotOptimizedExpr(glee.NewConstantExpr(0xBBAA, 16)),
			}, 8, 16)
			exp := &glee.ConcatExpr{
				MSB: &glee.ExtractExpr{Expr: glee.NewNotOptimizedExpr(glee.NewConstantExpr(0xDDCC, 16)), Offset: 0, Width: 8},
				LSB: &glee.ExtractExpr{Expr: glee.NewNotOptimizedExpr(glee.NewConstantExpr(0xBBAA, 16)), Offset: 8, Width: 8},
			}
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
	})
	t.Run("Symbolic", func(t *testing.T) {
		got := glee.NewExtractExpr(glee.NewNotOptimizedExpr(glee.NewConstantExpr(0xDDCC, 32)), 8, 16)
		exp := &glee.ExtractExpr{
			Expr:   glee.NewNotOptimizedExpr(glee.NewConstantExpr(0xDDCC, 32)),
			Offset: 8,
			Width:  16,
		}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestExtractExpr_String(t *testing.T) {
	expr := &glee.ExtractExpr{Expr: glee.NewConstantExpr(0, 32), Offset: 8, Width: 16}
	if s := expr.String(); s != "(extract (const 0 32) 8 16)" {
		t.Fatalf("unexpected string: %s", s)
	}
}

func TestNewNotExpr(t *testing.T) {
	t.Run("Constant", func(t *testing.T) {
		got := glee.NewNotExpr(glee.NewConstantExpr(0, 1))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Symbolic", func(t *testing.T) {
		got := glee.NewNotExpr(glee.NewNotOptimizedExpr(glee.NewConstantExpr(0xFFFF, 32)))
		exp := &glee.NotExpr{Expr: glee.NewNotOptimizedExpr(glee.NewConstantExpr(0xFFFF, 32))}
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestNotExpr_String(t *testing.T) {
	expr := &glee.NotExpr{Expr: glee.NewConstantExpr(0, 32)}
	if s := expr.String(); s != "(not (const 0 32))" {
		t.Fatalf("unexpected string: %s", s)
	}
}

func TestNewCastExpr(t *testing.T) {
	t.Run("Signed", func(t *testing.T) {
		t.Run("SameWidth", func(t *testing.T) {
			x := int16(-1000)
			got := glee.NewCastExpr(glee.NewConstantExpr(uint64(uint16(x)), 16), 16, true)
			exp := glee.NewConstantExpr(uint64(uint32(x)), 16)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
		t.Run("Truncate", func(t *testing.T) {
			x := int16(-1000)
			got := glee.NewCastExpr(glee.NewConstantExpr(uint64(uint16(x)), 16), 8, true)
			exp := glee.NewConstantExpr(24, 8)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
		t.Run("Constant", func(t *testing.T) {
			x := int16(-1000)
			got := glee.NewCastExpr(glee.NewConstantExpr(uint64(uint16(x)), 16), 32, true)
			exp := glee.NewConstantExpr(uint64(uint32(int32(x))), 32)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
		t.Run("Symbolic", func(t *testing.T) {
			got := glee.NewCastExpr(glee.NewNotOptimizedExpr(glee.NewConstantExpr(0, 16)), 32, true)
			exp := &glee.CastExpr{
				Src:    glee.NewNotOptimizedExpr(glee.NewConstantExpr(0, 16)),
				Width:  32,
				Signed: true,
			}
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
	})
	t.Run("Unsigned", func(t *testing.T) {
		t.Run("SameWidth", func(t *testing.T) {
			got := glee.NewCastExpr(glee.NewConstantExpr(1000, 16), 16, false)
			exp := glee.NewConstantExpr(1000, 16)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
		t.Run("Truncate", func(t *testing.T) {
			got := glee.NewCastExpr(glee.NewConstantExpr(1000, 16), 8, false)
			exp := glee.NewConstantExpr(1000, 8)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
		t.Run("Constant", func(t *testing.T) {
			got := glee.NewCastExpr(glee.NewConstantExpr(1000, 16), 32, false)
			exp := glee.NewConstantExpr(1000, 32)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
		t.Run("Symbolic", func(t *testing.T) {
			got := glee.NewCastExpr(glee.NewNotOptimizedExpr(glee.NewConstantExpr(0, 16)), 32, false)
			exp := &glee.CastExpr{
				Src:    glee.NewNotOptimizedExpr(glee.NewConstantExpr(0, 16)),
				Width:  32,
				Signed: false,
			}
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
	})
}

func TestCastExpr_String(t *testing.T) {
	t.Run("Signed", func(t *testing.T) {
		expr := &glee.CastExpr{Src: glee.NewConstantExpr(0, 16), Width: 32, Signed: true}
		if s := expr.String(); s != "(sext (const 0 16) 32)" {
			t.Fatalf("unexpected string: %s", s)
		}
	})
	t.Run("Signed", func(t *testing.T) {
		expr := &glee.CastExpr{Src: glee.NewConstantExpr(0, 16), Width: 32, Signed: false}
		if s := expr.String(); s != "(zext (const 0 16) 32)" {
			t.Fatalf("unexpected string: %s", s)
		}
	})
}

func TestConstantExpr_IsTrue(t *testing.T) {
	t.Run("Bool", func(t *testing.T) {
		t.Run("True", func(t *testing.T) {
			if !glee.NewConstantExpr(1, 1).IsTrue() {
				t.Fatal("expected true")
			}
		})
		t.Run("False", func(t *testing.T) {
			if glee.NewConstantExpr(0, 1).IsTrue() {
				t.Fatal("expected false")
			}
		})
	})
	t.Run("NonBool", func(t *testing.T) {
		if glee.NewConstantExpr(1, 8).IsTrue() {
			t.Fatal("expected false")
		}
	})
}

func TestConstantExpr_IsFalse(t *testing.T) {
	t.Run("Bool", func(t *testing.T) {
		t.Run("True", func(t *testing.T) {
			if glee.NewConstantExpr(1, 1).IsFalse() {
				t.Fatal("expected false")
			}
		})
		t.Run("False", func(t *testing.T) {
			if !glee.NewConstantExpr(0, 1).IsFalse() {
				t.Fatal("expected true")
			}
		})
	})
	t.Run("NonBool", func(t *testing.T) {
		if glee.NewConstantExpr(1, 8).IsFalse() {
			t.Fatal("expected false")
		}
	})
}

func TestConstantExpr_ZExt(t *testing.T) {
	t.Run("SameWidth", func(t *testing.T) {
		got := glee.NewConstantExpr(100, 32).ZExt(32)
		exp := glee.NewConstantExpr(100, 32)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Bool", func(t *testing.T) {
		got := glee.NewConstantExpr(100, 16).ZExt(1)
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("Extend", func(t *testing.T) {
		got := glee.NewConstantExpr(100, 16).ZExt(32)
		exp := glee.NewConstantExpr(100, 32)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestConstantExpr_SExt(t *testing.T) {
	t.Run("SameWidth", func(t *testing.T) {
		i32 := int32(-100)
		got := glee.NewConstantExpr(uint64(uint32(i32)), 32).SExt(32)
		exp := glee.NewConstantExpr(uint64(uint32(i32)), 32)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("8", func(t *testing.T) {
		t.Run("16", func(t *testing.T) {
			i8, i16 := int8(-100), int16(-100)
			got := glee.NewConstantExpr(uint64(uint8(i8)), 8).SExt(16)
			exp := glee.NewConstantExpr(uint64(uint16(i16)), 16)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
		t.Run("32", func(t *testing.T) {
			i8, i32 := int8(-100), int32(-100)
			got := glee.NewConstantExpr(uint64(uint8(i8)), 8).SExt(32)
			exp := glee.NewConstantExpr(uint64(uint32(i32)), 32)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
		t.Run("64", func(t *testing.T) {
			i8, i64 := int8(-100), int64(-100)
			got := glee.NewConstantExpr(uint64(uint8(i8)), 8).SExt(64)
			exp := glee.NewConstantExpr(uint64(uint64(i64)), 64)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
	})
	t.Run("16", func(t *testing.T) {
		t.Run("8", func(t *testing.T) {
			i16 := int16(-100)
			got := glee.NewConstantExpr(uint64(uint16(i16)), 16).SExt(8)
			exp := glee.NewConstantExpr(uint64(uint8(int8(i16))), 8)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
		t.Run("32", func(t *testing.T) {
			i16, i32 := int16(-100), int32(-100)
			got := glee.NewConstantExpr(uint64(uint16(i16)), 16).SExt(32)
			exp := glee.NewConstantExpr(uint64(uint32(i32)), 32)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
		t.Run("64", func(t *testing.T) {
			i16, i64 := int16(-100), int64(-100)
			got := glee.NewConstantExpr(uint64(uint16(i16)), 16).SExt(64)
			exp := glee.NewConstantExpr(uint64(uint64(i64)), 64)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
	})
	t.Run("32", func(t *testing.T) {
		t.Run("8", func(t *testing.T) {
			i32 := int32(-100)
			got := glee.NewConstantExpr(uint64(uint32(i32)), 32).SExt(8)
			exp := glee.NewConstantExpr(uint64(uint8(int8(i32))), 8)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
		t.Run("16", func(t *testing.T) {
			i32 := int32(-100)
			got := glee.NewConstantExpr(uint64(uint32(i32)), 32).SExt(16)
			exp := glee.NewConstantExpr(uint64(uint16(int16(i32))), 16)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
		t.Run("64", func(t *testing.T) {
			i32, i64 := int32(-100), int64(-100)
			got := glee.NewConstantExpr(uint64(uint32(i32)), 32).SExt(64)
			exp := glee.NewConstantExpr(uint64(uint64(i64)), 64)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
	})
	t.Run("64", func(t *testing.T) {
		t.Run("8", func(t *testing.T) {
			i64 := int64(-100)
			got := glee.NewConstantExpr(uint64(uint64(i64)), 64).SExt(8)
			exp := glee.NewConstantExpr(uint64(uint8(int8(i64))), 8)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
		t.Run("16", func(t *testing.T) {
			i64 := int64(-100)
			got := glee.NewConstantExpr(uint64(uint64(i64)), 64).SExt(16)
			exp := glee.NewConstantExpr(uint64(uint16(int16(i64))), 16)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
		t.Run("32", func(t *testing.T) {
			i64 := int64(-100)
			got := glee.NewConstantExpr(uint64(uint64(i64)), 64).SExt(32)
			exp := glee.NewConstantExpr(uint64(uint32(int32(i64))), 32)
			if diff := cmp.Diff(got, exp); diff != "" {
				t.Fatal(diff)
			}
		})
	})
}

func TestConstantExpr_UDiv(t *testing.T) {
	t.Run("8", func(t *testing.T) {
		got := glee.NewConstantExpr(100, 8).UDiv(glee.NewConstantExpr(20, 8))
		exp := glee.NewConstantExpr(5, 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("16", func(t *testing.T) {
		got := glee.NewConstantExpr(100, 16).UDiv(glee.NewConstantExpr(20, 16))
		exp := glee.NewConstantExpr(5, 16)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("32", func(t *testing.T) {
		got := glee.NewConstantExpr(100, 32).UDiv(glee.NewConstantExpr(20, 32))
		exp := glee.NewConstantExpr(5, 32)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("64", func(t *testing.T) {
		got := glee.NewConstantExpr(100, 64).UDiv(glee.NewConstantExpr(20, 64))
		exp := glee.NewConstantExpr(5, 64)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestConstantExpr_SDiv(t *testing.T) {
	t.Run("8", func(t *testing.T) {
		x, y := int8(-100), int8(-5)
		got := glee.NewConstantExpr(uint64(uint8(x)), 8).SDiv(glee.NewConstantExpr(20, 8))
		exp := glee.NewConstantExpr(uint64(uint8(y)), 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("16", func(t *testing.T) {
		x, y := int16(-100), int16(-5)
		got := glee.NewConstantExpr(uint64(uint16(x)), 16).SDiv(glee.NewConstantExpr(20, 16))
		exp := glee.NewConstantExpr(uint64(uint16(y)), 16)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("32", func(t *testing.T) {
		x, y := int32(-100), int32(-5)
		got := glee.NewConstantExpr(uint64(uint32(x)), 32).SDiv(glee.NewConstantExpr(20, 32))
		exp := glee.NewConstantExpr(uint64(uint32(y)), 32)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("64", func(t *testing.T) {
		x, y := int64(-100), int64(-5)
		got := glee.NewConstantExpr(uint64(uint64(x)), 64).SDiv(glee.NewConstantExpr(20, 64))
		exp := glee.NewConstantExpr(uint64(uint64(y)), 64)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestConstantExpr_URem(t *testing.T) {
	t.Run("8", func(t *testing.T) {
		got := glee.NewConstantExpr(100, 8).URem(glee.NewConstantExpr(7, 8))
		exp := glee.NewConstantExpr(2, 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("16", func(t *testing.T) {
		got := glee.NewConstantExpr(100, 16).URem(glee.NewConstantExpr(7, 16))
		exp := glee.NewConstantExpr(2, 16)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("32", func(t *testing.T) {
		got := glee.NewConstantExpr(100, 32).URem(glee.NewConstantExpr(7, 32))
		exp := glee.NewConstantExpr(2, 32)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("64", func(t *testing.T) {
		got := glee.NewConstantExpr(100, 64).URem(glee.NewConstantExpr(7, 64))
		exp := glee.NewConstantExpr(2, 64)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestConstantExpr_SRem(t *testing.T) {
	t.Run("8", func(t *testing.T) {
		x, y := int8(-100), int8(-2)
		got := glee.NewConstantExpr(uint64(uint8(x)), 8).SRem(glee.NewConstantExpr(7, 8))
		exp := glee.NewConstantExpr(uint64(uint8(y)), 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("16", func(t *testing.T) {
		x, y := int16(-100), int16(-2)
		got := glee.NewConstantExpr(uint64(uint16(x)), 16).SRem(glee.NewConstantExpr(7, 16))
		exp := glee.NewConstantExpr(uint64(uint16(y)), 16)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("32", func(t *testing.T) {
		x, y := int32(-100), int32(-2)
		got := glee.NewConstantExpr(uint64(uint32(x)), 32).SRem(glee.NewConstantExpr(7, 32))
		exp := glee.NewConstantExpr(uint64(uint32(y)), 32)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("64", func(t *testing.T) {
		x, y := int64(-100), int64(-2)
		got := glee.NewConstantExpr(uint64(uint64(x)), 64).SRem(glee.NewConstantExpr(7, 64))
		exp := glee.NewConstantExpr(uint64(uint64(y)), 64)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestConstantExpr_And(t *testing.T) {
	got := glee.NewConstantExpr(0x0FF0, 16).And(glee.NewConstantExpr(0xFF0F, 16))
	exp := glee.NewConstantExpr(0x0F00, 16)
	if diff := cmp.Diff(got, exp); diff != "" {
		t.Fatal(diff)
	}
}

func TestConstantExpr_Or(t *testing.T) {
	got := glee.NewConstantExpr(0x00F0, 16).Or(glee.NewConstantExpr(0xFF00, 16))
	exp := glee.NewConstantExpr(0xFFF0, 16)
	if diff := cmp.Diff(got, exp); diff != "" {
		t.Fatal(diff)
	}
}

func TestConstantExpr_Xor(t *testing.T) {
	got := glee.NewConstantExpr(0x0FF0, 16).Xor(glee.NewConstantExpr(0xFF00, 16))
	exp := glee.NewConstantExpr(0xF0F0, 16)
	if diff := cmp.Diff(got, exp); diff != "" {
		t.Fatal(diff)
	}
}

func TestConstantExpr_Shl(t *testing.T) {
	t.Run("8", func(t *testing.T) {
		got := glee.NewConstantExpr(0xF3, 8).Shl(glee.NewConstantExpr(4, 16))
		exp := glee.NewConstantExpr(0x30, 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("16", func(t *testing.T) {
		got := glee.NewConstantExpr(0xF3, 16).Shl(glee.NewConstantExpr(4, 16))
		exp := glee.NewConstantExpr(0x0F30, 16)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("32", func(t *testing.T) {
		got := glee.NewConstantExpr(0xF3, 32).Shl(glee.NewConstantExpr(4, 16))
		exp := glee.NewConstantExpr(0x0F30, 32)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("64", func(t *testing.T) {
		got := glee.NewConstantExpr(0xF3, 64).Shl(glee.NewConstantExpr(4, 16))
		exp := glee.NewConstantExpr(0x0F30, 64)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestConstantExpr_LShr(t *testing.T) {
	t.Run("8", func(t *testing.T) {
		got := glee.NewConstantExpr(0xF3, 8).LShr(glee.NewConstantExpr(4, 16))
		exp := glee.NewConstantExpr(0x0F, 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("16", func(t *testing.T) {
		got := glee.NewConstantExpr(0xF3, 16).LShr(glee.NewConstantExpr(4, 16))
		exp := glee.NewConstantExpr(0x0F, 16)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("32", func(t *testing.T) {
		got := glee.NewConstantExpr(0xF3, 32).LShr(glee.NewConstantExpr(4, 16))
		exp := glee.NewConstantExpr(0x0F, 32)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("64", func(t *testing.T) {
		got := glee.NewConstantExpr(0xF3, 64).LShr(glee.NewConstantExpr(4, 16))
		exp := glee.NewConstantExpr(0x0F, 64)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestConstantExpr_AShr(t *testing.T) {
	t.Run("8", func(t *testing.T) {
		got := glee.NewConstantExpr(0xF0, 8).AShr(glee.NewConstantExpr(4, 16))
		exp := glee.NewConstantExpr(0xFF, 8)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("16", func(t *testing.T) {
		got := glee.NewConstantExpr(0x7000, 16).AShr(glee.NewConstantExpr(4, 16))
		exp := glee.NewConstantExpr(0x0700, 16)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("32", func(t *testing.T) {
		got := glee.NewConstantExpr(0xF0, 32).AShr(glee.NewConstantExpr(4, 16))
		exp := glee.NewConstantExpr(0x0F, 32)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("64", func(t *testing.T) {
		got := glee.NewConstantExpr(0XFFFFFFFF00000000, 64).AShr(glee.NewConstantExpr(4, 16))
		exp := glee.NewConstantExpr(0XFFFFFFFFF0000000, 64)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestConstantExpr_Eq(t *testing.T) {
	t.Run("True", func(t *testing.T) {
		got := glee.NewConstantExpr(100, 8).Eq(glee.NewConstantExpr(100, 8))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("False", func(t *testing.T) {
		got := glee.NewConstantExpr(3, 8).Eq(glee.NewConstantExpr(100, 8))
		exp := glee.NewConstantExpr(0, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestConstantExpr_Ult(t *testing.T) {
	t.Run("8", func(t *testing.T) {
		got := glee.NewConstantExpr(100, 8).Ult(glee.NewConstantExpr(120, 8))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("16", func(t *testing.T) {
		got := glee.NewConstantExpr(100, 16).Ult(glee.NewConstantExpr(120, 16))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("32", func(t *testing.T) {
		got := glee.NewConstantExpr(100, 32).Ult(glee.NewConstantExpr(120, 32))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("64", func(t *testing.T) {
		got := glee.NewConstantExpr(100, 64).Ult(glee.NewConstantExpr(120, 64))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestConstantExpr_Ugt(t *testing.T) {
	got := glee.NewConstantExpr(120, 8).Ugt(glee.NewConstantExpr(100, 8))
	exp := glee.NewConstantExpr(1, 1)
	if diff := cmp.Diff(got, exp); diff != "" {
		t.Fatal(diff)
	}
}

func TestConstantExpr_Ule(t *testing.T) {
	t.Run("8", func(t *testing.T) {
		got := glee.NewConstantExpr(100, 8).Ule(glee.NewConstantExpr(120, 8))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("16", func(t *testing.T) {
		got := glee.NewConstantExpr(100, 16).Ule(glee.NewConstantExpr(120, 16))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("32", func(t *testing.T) {
		got := glee.NewConstantExpr(100, 32).Ule(glee.NewConstantExpr(120, 32))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("64", func(t *testing.T) {
		got := glee.NewConstantExpr(100, 64).Ule(glee.NewConstantExpr(120, 64))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestConstantExpr_Uge(t *testing.T) {
	got := glee.NewConstantExpr(120, 8).Uge(glee.NewConstantExpr(100, 8))
	exp := glee.NewConstantExpr(1, 1)
	if diff := cmp.Diff(got, exp); diff != "" {
		t.Fatal(diff)
	}
}

func TestConstantExpr_Slt(t *testing.T) {
	t.Run("8", func(t *testing.T) {
		x := int8(-100)
		got := glee.NewConstantExpr(uint64(uint8(x)), 8).Slt(glee.NewConstantExpr(120, 8))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("16", func(t *testing.T) {
		x := int16(-100)
		got := glee.NewConstantExpr(uint64(uint16(x)), 16).Slt(glee.NewConstantExpr(120, 16))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("32", func(t *testing.T) {
		x := int32(-100)
		got := glee.NewConstantExpr(uint64(uint32(x)), 32).Slt(glee.NewConstantExpr(120, 32))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("64", func(t *testing.T) {
		x := int64(-100)
		got := glee.NewConstantExpr(uint64(x), 64).Slt(glee.NewConstantExpr(120, 64))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestConstantExpr_Sgt(t *testing.T) {
	x := int8(-100)
	got := glee.NewConstantExpr(120, 8).Sgt(glee.NewConstantExpr(uint64(uint8(x)), 8))
	exp := glee.NewConstantExpr(1, 1)
	if diff := cmp.Diff(got, exp); diff != "" {
		t.Fatal(diff)
	}
}

func TestConstantExpr_Sle(t *testing.T) {
	t.Run("8", func(t *testing.T) {
		x := int8(-100)
		got := glee.NewConstantExpr(uint64(uint8(x)), 8).Sle(glee.NewConstantExpr(120, 8))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("16", func(t *testing.T) {
		x := int16(-100)
		got := glee.NewConstantExpr(uint64(uint16(x)), 16).Sle(glee.NewConstantExpr(120, 16))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("32", func(t *testing.T) {
		x := int32(-100)
		got := glee.NewConstantExpr(uint64(uint32(x)), 32).Sle(glee.NewConstantExpr(120, 32))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("64", func(t *testing.T) {
		x := int64(-100)
		got := glee.NewConstantExpr(uint64(x), 64).Sle(glee.NewConstantExpr(120, 64))
		exp := glee.NewConstantExpr(1, 1)
		if diff := cmp.Diff(got, exp); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestConstantExpr_Sge(t *testing.T) {
	x := int8(-100)
	got := glee.NewConstantExpr(120, 8).Sge(glee.NewConstantExpr(uint64(uint8(x)), 8))
	exp := glee.NewConstantExpr(1, 1)
	if diff := cmp.Diff(got, exp); diff != "" {
		t.Fatal(diff)
	}
}

func TestIsConstantTrue(t *testing.T) {
	t.Run("Bool", func(t *testing.T) {
		t.Run("True", func(t *testing.T) {
			if !glee.IsConstantTrue(glee.NewConstantExpr(1, 1)) {
				t.Fatal("expected true")
			}
		})
		t.Run("False", func(t *testing.T) {
			if glee.IsConstantTrue(glee.NewConstantExpr(0, 1)) {
				t.Fatal("expected false")
			}
		})
	})
	t.Run("NonBool", func(t *testing.T) {
		if glee.IsConstantTrue(glee.NewConstantExpr(1, 8)) {
			t.Fatal("expected false")
		}
	})
}

func TestIsConstantFalse(t *testing.T) {
	t.Run("Bool", func(t *testing.T) {
		t.Run("True", func(t *testing.T) {
			if glee.IsConstantFalse(glee.NewConstantExpr(1, 1)) {
				t.Fatal("expected false")
			}
		})
		t.Run("False", func(t *testing.T) {
			if !glee.IsConstantFalse(glee.NewConstantExpr(0, 1)) {
				t.Fatal("expected true")
			}
		})
	})
	t.Run("NonBool", func(t *testing.T) {
		if glee.IsConstantFalse(glee.NewConstantExpr(1, 8)) {
			t.Fatal("expected false")
		}
	})
}

func TestNewNotOptimizedExpr(t *testing.T) {
	got := glee.NewNotOptimizedExpr(glee.NewConstantExpr(0, 1))
	exp := &glee.NotOptimizedExpr{Src: glee.NewConstantExpr(0, 1)}
	if diff := cmp.Diff(got, exp); diff != "" {
		t.Fatal(diff)
	}
}

func TestNotOptimizedExpr_String(t *testing.T) {
	expr := &glee.NotOptimizedExpr{Src: glee.NewConstantExpr(0, 32)}
	if s := expr.String(); s != "(no-opt (const 0 32))" {
		t.Fatalf("unexpected string: %s", s)
	}
}

func TestTuple_String(t *testing.T) {
	expr := glee.Tuple{
		glee.NewConstantExpr(0, 32),
		glee.NewConstantExpr(1, 32),
	}
	if s := expr.String(); s != "[(const 0 32) (const 1 32)]" {
		t.Fatalf("unexpected string: %s", s)
	}
}
