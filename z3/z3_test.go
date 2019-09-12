package z3_test

import (
	"testing"

	"github.com/benbjohnson/glee"
	"github.com/benbjohnson/glee/z3"
	"github.com/google/go-cmp/cmp"
)

func TestSolver_Solve(t *testing.T) {
	t.Run("Constant", func(t *testing.T) {
		t.Run("True", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)
			if satisfiable, _, err := s.Solve([]glee.Expr{glee.NewBoolConstantExpr(true)}, nil); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			}
		})
		t.Run("False", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)
			if satisfiable, _, err := s.Solve([]glee.Expr{glee.NewBoolConstantExpr(false)}, nil); err != nil {
				t.Fatal(err)
			} else if satisfiable {
				t.Fatal("expected unsatisfiable")
			}
		})
	})

	t.Run("Array", func(t *testing.T) {
		t.Run("Width8", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)

			array := glee.NewArray(100, 1)

			if satisfiable, values, err := s.Solve(
				[]glee.Expr{
					glee.NewBinaryExpr(glee.EQ,
						array.Select(glee.NewConstantExpr(0, 64), 8, false),
						glee.NewConstantExpr(10, 8),
					),
				},
				[]*glee.Array{array},
			); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			} else if diff := cmp.Diff(values, [][]byte{{10}}); diff != "" {
				t.Fatal(diff)
			}
		})
		t.Run("Width16", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)

			array := glee.NewArray(100, 2)

			if satisfiable, values, err := s.Solve(
				[]glee.Expr{
					glee.NewBinaryExpr(glee.EQ,
						array.Select(glee.NewConstantExpr(0, 64), 16, false),
						glee.NewConstantExpr(0xAABB, 16),
					),
				},
				[]*glee.Array{array},
			); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			} else if diff := cmp.Diff(values, [][]byte{{0xAA, 0xBB}}); diff != "" {
				t.Fatal(diff)
			}
		})
	})

	t.Run("NotOptimized", func(t *testing.T) {
		s := z3.NewSolver()
		defer MustCloseSolver(s)
		if satisfiable, _, err := s.Solve([]glee.Expr{glee.NewNotOptimizedExpr(glee.NewBoolConstantExpr(true))}, nil); err != nil {
			t.Fatal(err)
		} else if !satisfiable {
			t.Fatal("expected satisfiable")
		}
	})

	t.Run("Extract", func(t *testing.T) {
		t.Run("Bool", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)

			// Extract 1 bit
			if satisfiable, _, err := s.Solve([]glee.Expr{
				&glee.ExtractExpr{
					Expr:   glee.NewConstantExpr(0x04, 64),
					Offset: 2,
					Width:  1,
				},
			}, nil); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			}

			// Extract 0 bit.
			if satisfiable, _, err := s.Solve([]glee.Expr{
				&glee.ExtractExpr{
					Expr:   glee.NewConstantExpr(0x04, 64),
					Offset: 6,
					Width:  1,
				},
			}, nil); err != nil {
				t.Fatal(err)
			} else if satisfiable {
				t.Fatal("expected unsatisfiable")
			}
		})
		t.Run("Int", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)
			if satisfiable, _, err := s.Solve([]glee.Expr{
				&glee.BinaryExpr{
					Op: glee.EQ,
					LHS: &glee.ExtractExpr{
						Expr:   glee.NewConstantExpr(0xAABB, 16),
						Offset: 8,
						Width:  8,
					},
					RHS: glee.NewConstantExpr(0xAA, 8),
				},
			}, nil); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			}
		})
	})

	t.Run("Cast", func(t *testing.T) {
		t.Run("Signed", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)

			value := -200
			if satisfiable, _, err := s.Solve([]glee.Expr{
				&glee.BinaryExpr{
					Op: glee.EQ,
					LHS: &glee.CastExpr{
						Src:    glee.NewConstantExpr(uint64(uint16(int16(value))), 16),
						Width:  32,
						Signed: true,
					},
					RHS: glee.NewConstantExpr(uint64(uint32(int32(value))), 32),
				},
			}, nil); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			}
		})
		t.Run("SignedBool", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)
			value := -1
			if satisfiable, _, err := s.Solve([]glee.Expr{
				&glee.BinaryExpr{
					Op: glee.EQ,
					LHS: &glee.CastExpr{
						Src:    glee.NewBoolConstantExpr(true),
						Width:  16,
						Signed: true,
					},
					RHS: glee.NewConstantExpr(uint64(uint16(int16(value))), 16),
				},
			}, nil); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			}
		})

		t.Run("Unsigned", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)
			if satisfiable, _, err := s.Solve([]glee.Expr{
				&glee.BinaryExpr{
					Op: glee.EQ,
					LHS: &glee.CastExpr{
						Src:   glee.NewConstantExpr(200, 16),
						Width: 32,
					},
					RHS: glee.NewConstantExpr(200, 32),
				},
			}, nil); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			}
		})
		t.Run("UnsignedBool", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)
			if satisfiable, _, err := s.Solve([]glee.Expr{
				&glee.BinaryExpr{
					Op: glee.EQ,
					LHS: &glee.CastExpr{
						Src:   glee.NewBoolConstantExpr(true),
						Width: 16,
					},
					RHS: glee.NewConstantExpr(1, 16),
				},
			}, nil); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			}
		})
	})

	t.Run("Not", func(t *testing.T) {
		t.Run("Bool", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)
			if satisfiable, _, err := s.Solve([]glee.Expr{
				&glee.BinaryExpr{
					Op: glee.EQ,
					LHS: &glee.NotExpr{
						Expr: glee.NewBoolConstantExpr(true),
					},
					RHS: glee.NewBoolConstantExpr(false),
				},
			}, nil); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			}
		})
		t.Run("Int", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)
			if satisfiable, _, err := s.Solve([]glee.Expr{
				&glee.BinaryExpr{
					Op: glee.EQ,
					LHS: &glee.NotExpr{
						Expr: glee.NewConstantExpr(0xFF00FF00, 16),
					},
					RHS: glee.NewConstantExpr(0x00FF00FF, 16),
				},
			}, nil); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			}
		})
	})

	t.Run("BinaryExpr", func(t *testing.T) {
		t.Run("ADD", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)
			if satisfiable, _, err := s.Solve([]glee.Expr{
				&glee.BinaryExpr{
					Op: glee.EQ,
					LHS: &glee.BinaryExpr{
						Op:  glee.ADD,
						LHS: glee.NewConstantExpr(1000, 16),
						RHS: glee.NewConstantExpr(200, 16),
					},
					RHS: glee.NewConstantExpr(1200, 16),
				},
			}, nil); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			}
		})
		t.Run("SUB", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)
			if satisfiable, _, err := s.Solve([]glee.Expr{
				&glee.BinaryExpr{
					Op: glee.EQ,
					LHS: &glee.BinaryExpr{
						Op:  glee.SUB,
						LHS: glee.NewConstantExpr(1000, 16),
						RHS: glee.NewConstantExpr(200, 16),
					},
					RHS: glee.NewConstantExpr(800, 16),
				},
			}, nil); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			}
		})
		t.Run("MUL", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)
			if satisfiable, _, err := s.Solve([]glee.Expr{
				&glee.BinaryExpr{
					Op: glee.EQ,
					LHS: &glee.BinaryExpr{
						Op:  glee.MUL,
						LHS: glee.NewConstantExpr(30, 16),
						RHS: glee.NewConstantExpr(200, 16),
					},
					RHS: glee.NewConstantExpr(6000, 16),
				},
			}, nil); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			}
		})
		t.Run("UDIV", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)
			if satisfiable, _, err := s.Solve([]glee.Expr{
				&glee.BinaryExpr{
					Op: glee.EQ,
					LHS: &glee.BinaryExpr{
						Op:  glee.UDIV,
						LHS: glee.NewConstantExpr(5000, 16),
						RHS: glee.NewConstantExpr(30, 16),
					},
					RHS: glee.NewConstantExpr(166, 16),
				},
			}, nil); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			}
		})
		t.Run("SDIV", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)
			x, y := -30, -166
			if satisfiable, _, err := s.Solve([]glee.Expr{
				&glee.BinaryExpr{
					Op: glee.EQ,
					LHS: &glee.BinaryExpr{
						Op:  glee.SDIV,
						LHS: glee.NewConstantExpr(5000, 16),
						RHS: glee.NewConstantExpr(uint64(uint16(int16(x))), 16),
					},
					RHS: glee.NewConstantExpr(uint64(uint16(int16(y))), 16),
				},
			}, nil); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			}
		})
		t.Run("UREM", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)
			if satisfiable, _, err := s.Solve([]glee.Expr{
				&glee.BinaryExpr{
					Op: glee.EQ,
					LHS: &glee.BinaryExpr{
						Op:  glee.UREM,
						LHS: glee.NewConstantExpr(5000, 16),
						RHS: glee.NewConstantExpr(30, 16),
					},
					RHS: glee.NewConstantExpr(20, 16),
				},
			}, nil); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			}
		})
		t.Run("SREM", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)
			x, y := -30, 20
			if satisfiable, _, err := s.Solve([]glee.Expr{
				&glee.BinaryExpr{
					Op: glee.EQ,
					LHS: &glee.BinaryExpr{
						Op:  glee.SREM,
						LHS: glee.NewConstantExpr(5000, 16),
						RHS: glee.NewConstantExpr(uint64(uint16(int16(x))), 16),
					},
					RHS: glee.NewConstantExpr(uint64(uint16(int16(y))), 16),
				},
			}, nil); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			}
		})
		t.Run("AND", func(t *testing.T) {
			t.Run("Bool", func(t *testing.T) {
				s := z3.NewSolver()
				defer MustCloseSolver(s)
				if satisfiable, _, err := s.Solve([]glee.Expr{
					&glee.BinaryExpr{
						Op: glee.EQ,
						LHS: &glee.BinaryExpr{
							Op:  glee.AND,
							LHS: glee.NewBoolConstantExpr(true),
							RHS: glee.NewBoolConstantExpr(true),
						},
						RHS: glee.NewBoolConstantExpr(true),
					},
				}, nil); err != nil {
					t.Fatal(err)
				} else if !satisfiable {
					t.Fatal("expected satisfiable")
				}
			})
			t.Run("Int", func(t *testing.T) {
				s := z3.NewSolver()
				defer MustCloseSolver(s)
				if satisfiable, _, err := s.Solve([]glee.Expr{
					&glee.BinaryExpr{
						Op: glee.EQ,
						LHS: &glee.BinaryExpr{
							Op:  glee.AND,
							LHS: glee.NewConstantExpr(0x0FF0, 16),
							RHS: glee.NewConstantExpr(0xFF00, 16),
						},
						RHS: glee.NewConstantExpr(0x0F00, 16),
					},
				}, nil); err != nil {
					t.Fatal(err)
				} else if !satisfiable {
					t.Fatal("expected satisfiable")
				}
			})
		})
		t.Run("OR", func(t *testing.T) {
			t.Run("Bool", func(t *testing.T) {
				s := z3.NewSolver()
				defer MustCloseSolver(s)
				if satisfiable, _, err := s.Solve([]glee.Expr{
					&glee.BinaryExpr{
						Op: glee.EQ,
						LHS: &glee.BinaryExpr{
							Op:  glee.OR,
							LHS: glee.NewBoolConstantExpr(true),
							RHS: glee.NewBoolConstantExpr(false),
						},
						RHS: glee.NewBoolConstantExpr(true),
					},
				}, nil); err != nil {
					t.Fatal(err)
				} else if !satisfiable {
					t.Fatal("expected satisfiable")
				}
			})
			t.Run("Int", func(t *testing.T) {
				s := z3.NewSolver()
				defer MustCloseSolver(s)
				if satisfiable, _, err := s.Solve([]glee.Expr{
					&glee.BinaryExpr{
						Op: glee.EQ,
						LHS: &glee.BinaryExpr{
							Op:  glee.OR,
							LHS: glee.NewConstantExpr(0x0FF0, 16),
							RHS: glee.NewConstantExpr(0xFF00, 16),
						},
						RHS: glee.NewConstantExpr(0xFFF0, 16),
					},
				}, nil); err != nil {
					t.Fatal(err)
				} else if !satisfiable {
					t.Fatal("expected satisfiable")
				}
			})
		})
		t.Run("XOR", func(t *testing.T) {
			t.Run("Bool", func(t *testing.T) {
				s := z3.NewSolver()
				defer MustCloseSolver(s)
				if satisfiable, _, err := s.Solve([]glee.Expr{
					&glee.BinaryExpr{
						Op: glee.EQ,
						LHS: &glee.BinaryExpr{
							Op:  glee.XOR,
							LHS: glee.NewBoolConstantExpr(true),
							RHS: glee.NewBoolConstantExpr(true),
						},
						RHS: glee.NewBoolConstantExpr(false),
					},
				}, nil); err != nil {
					t.Fatal(err)
				} else if !satisfiable {
					t.Fatal("expected satisfiable")
				}
			})
			t.Run("Int", func(t *testing.T) {
				s := z3.NewSolver()
				defer MustCloseSolver(s)
				if satisfiable, _, err := s.Solve([]glee.Expr{
					&glee.BinaryExpr{
						Op: glee.EQ,
						LHS: &glee.BinaryExpr{
							Op:  glee.XOR,
							LHS: glee.NewConstantExpr(0x0FF0, 16),
							RHS: glee.NewConstantExpr(0xFF00, 16),
						},
						RHS: glee.NewConstantExpr(0xF0F0, 16),
					},
				}, nil); err != nil {
					t.Fatal(err)
				} else if !satisfiable {
					t.Fatal("expected satisfiable")
				}
			})
		})
		t.Run("SHL", func(t *testing.T) {
			t.Run("Constant", func(t *testing.T) {
				s := z3.NewSolver()
				defer MustCloseSolver(s)
				if satisfiable, _, err := s.Solve([]glee.Expr{
					&glee.BinaryExpr{
						Op: glee.EQ,
						LHS: &glee.BinaryExpr{
							Op:  glee.SHL,
							LHS: glee.NewConstantExpr(0x0FF0, 16),
							RHS: glee.NewConstantExpr(4, 16),
						},
						RHS: glee.NewConstantExpr(0xFF00, 16),
					},
				}, nil); err != nil {
					t.Fatal(err)
				} else if !satisfiable {
					t.Fatal("expected satisfiable")
				}
			})
			t.Run("Symbolic", func(t *testing.T) {
				s := z3.NewSolver()
				defer MustCloseSolver(s)
				array := glee.NewArray(100, 2)
				if satisfiable, values, err := s.Solve([]glee.Expr{
					&glee.BinaryExpr{
						Op: glee.EQ,
						LHS: &glee.BinaryExpr{
							Op:  glee.SHL,
							LHS: glee.NewConstantExpr(0x0FF0, 16),
							RHS: array.Select(glee.NewConstantExpr64(0), 16, false),
						},
						RHS: glee.NewConstantExpr(0xFF00, 16),
					},
				},
					[]*glee.Array{array},
				); err != nil {
					t.Fatal(err)
				} else if !satisfiable {
					t.Fatal("expected satisfiable")
				} else if diff := cmp.Diff(values, [][]byte{{0x00, 0x04}}); diff != "" {
					t.Fatal(diff)
				}
			})
		})
		t.Run("LSHR", func(t *testing.T) {
			t.Run("Constant", func(t *testing.T) {
				s := z3.NewSolver()
				defer MustCloseSolver(s)
				if satisfiable, _, err := s.Solve([]glee.Expr{
					&glee.BinaryExpr{
						Op: glee.EQ,
						LHS: &glee.BinaryExpr{
							Op:  glee.LSHR,
							LHS: glee.NewConstantExpr(0x0FF0, 16),
							RHS: glee.NewConstantExpr(4, 16),
						},
						RHS: glee.NewConstantExpr(0x00FF, 16),
					},
				}, nil); err != nil {
					t.Fatal(err)
				} else if !satisfiable {
					t.Fatal("expected satisfiable")
				}
			})
			t.Run("Symbolic", func(t *testing.T) {
				s := z3.NewSolver()
				defer MustCloseSolver(s)
				array := glee.NewArray(100, 2)
				if satisfiable, values, err := s.Solve([]glee.Expr{
					&glee.BinaryExpr{
						Op: glee.EQ,
						LHS: &glee.BinaryExpr{
							Op:  glee.LSHR,
							LHS: glee.NewConstantExpr(0x0FF0, 16),
							RHS: array.Select(glee.NewConstantExpr64(0), 16, false),
						},
						RHS: glee.NewConstantExpr(0x00FF, 16),
					},
				},
					[]*glee.Array{array},
				); err != nil {
					t.Fatal(err)
				} else if !satisfiable {
					t.Fatal("expected satisfiable")
				} else if diff := cmp.Diff(values, [][]byte{{0x00, 0x04}}); diff != "" {
					t.Fatal(diff)
				}
			})
		})
		t.Run("ASHR", func(t *testing.T) {
			t.Run("Constant", func(t *testing.T) {
				s := z3.NewSolver()
				defer MustCloseSolver(s)
				if satisfiable, _, err := s.Solve([]glee.Expr{
					&glee.BinaryExpr{
						Op: glee.EQ,
						LHS: &glee.BinaryExpr{
							Op:  glee.ASHR,
							LHS: glee.NewConstantExpr(0x0FF0, 16),
							RHS: glee.NewConstantExpr(4, 16),
						},
						RHS: glee.NewConstantExpr(0x00FF, 16),
					},
				}, nil); err != nil {
					t.Fatal(err)
				} else if !satisfiable {
					t.Fatal("expected satisfiable")
				}
			})
			t.Run("Symbolic", func(t *testing.T) {
				s := z3.NewSolver()
				defer MustCloseSolver(s)
				array := glee.NewArray(100, 2)
				if satisfiable, values, err := s.Solve([]glee.Expr{
					&glee.BinaryExpr{
						Op: glee.EQ,
						LHS: &glee.BinaryExpr{
							Op:  glee.ASHR,
							LHS: glee.NewConstantExpr(0xFF00, 16),
							RHS: array.Select(glee.NewConstantExpr64(0), 16, false),
						},
						RHS: glee.NewConstantExpr(0xFFF0, 16),
					},
				},
					[]*glee.Array{array},
				); err != nil {
					t.Fatal(err)
				} else if !satisfiable {
					t.Fatal("expected satisfiable")
				} else if diff := cmp.Diff(values, [][]byte{{0x00, 0x04}}); diff != "" {
					t.Fatal(diff)
				}
			})
		})
		t.Run("EQ", func(t *testing.T) {
			t.Run("Bool", func(t *testing.T) {
				s := z3.NewSolver()
				defer MustCloseSolver(s)
				if satisfiable, _, err := s.Solve([]glee.Expr{
					&glee.BinaryExpr{
						Op:  glee.EQ,
						LHS: glee.NewBoolConstantExpr(true),
						RHS: glee.NewBoolConstantExpr(true),
					},
				}, nil); err != nil {
					t.Fatal(err)
				} else if !satisfiable {
					t.Fatal("expected satisfiable")
				}
			})
			t.Run("ConstantTrue", func(t *testing.T) {
				s := z3.NewSolver()
				defer MustCloseSolver(s)
				array := glee.NewArray(100, 1)
				if satisfiable, values, err := s.Solve([]glee.Expr{
					&glee.BinaryExpr{
						Op:  glee.EQ,
						LHS: glee.NewBoolConstantExpr(true),
						RHS: array.Select(glee.NewConstantExpr64(0), 1, false),
					},
				}, []*glee.Array{array}); err != nil {
					t.Fatal(err)
				} else if !satisfiable {
					t.Fatal("expected satisfiable")
				} else if diff := cmp.Diff(values, [][]byte{{0x01}}); diff != "" {
					t.Fatal(diff)
				}
			})
			t.Run("ConstantNotTrue", func(t *testing.T) {
				s := z3.NewSolver()
				defer MustCloseSolver(s)
				array := glee.NewArray(100, 1)
				if satisfiable, values, err := s.Solve([]glee.Expr{
					&glee.BinaryExpr{
						Op:  glee.EQ,
						LHS: glee.NewBoolConstantExpr(false),
						RHS: array.Select(glee.NewConstantExpr64(0), 1, false),
					},
				}, []*glee.Array{array}); err != nil {
					t.Fatal(err)
				} else if !satisfiable {
					t.Fatal("expected satisfiable")
				} else if diff := cmp.Diff(values, [][]byte{{0x00}}); diff != "" {
					t.Fatal(diff)
				}
			})
			t.Run("Int", func(t *testing.T) {
				s := z3.NewSolver()
				defer MustCloseSolver(s)
				if satisfiable, _, err := s.Solve([]glee.Expr{
					&glee.BinaryExpr{
						Op:  glee.EQ,
						LHS: glee.NewConstantExpr(10, 32),
						RHS: glee.NewConstantExpr(10, 32),
					},
				}, nil); err != nil {
					t.Fatal(err)
				} else if !satisfiable {
					t.Fatal("expected satisfiable")
				}
			})
		})
		t.Run("ULT", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)
			if satisfiable, _, err := s.Solve([]glee.Expr{
				&glee.BinaryExpr{
					Op:  glee.ULT,
					LHS: glee.NewConstantExpr(9, 32),
					RHS: glee.NewConstantExpr(10, 32),
				},
			}, nil); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			}
		})
		t.Run("ULE", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)
			if satisfiable, _, err := s.Solve([]glee.Expr{
				&glee.BinaryExpr{
					Op:  glee.ULE,
					LHS: glee.NewConstantExpr(10, 32),
					RHS: glee.NewConstantExpr(10, 32),
				},
			}, nil); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			}
		})
		t.Run("SLT", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)
			if satisfiable, _, err := s.Solve([]glee.Expr{
				&glee.BinaryExpr{
					Op:  glee.SLT,
					LHS: glee.NewConstantExpr(0xF0, 8),
					RHS: glee.NewConstantExpr(0x00, 8),
				},
			}, nil); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			}
		})
		t.Run("SLE", func(t *testing.T) {
			s := z3.NewSolver()
			defer MustCloseSolver(s)
			if satisfiable, _, err := s.Solve([]glee.Expr{
				&glee.BinaryExpr{
					Op:  glee.SLE,
					LHS: glee.NewConstantExpr(0xF0, 8),
					RHS: glee.NewConstantExpr(0xF0, 8),
				},
			}, nil); err != nil {
				t.Fatal(err)
			} else if !satisfiable {
				t.Fatal("expected satisfiable")
			}
		})
	})
}

func MustCloseSolver(s *z3.Solver) {
	if err := s.Close(); err != nil {
		panic(err)
	}
}
