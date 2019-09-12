package glee_test

import (
	"testing"

	"github.com/benbjohnson/glee"
	"github.com/google/go-cmp/cmp"
)

func TestArray(t *testing.T) {
	t.Run("Concrete", func(t *testing.T) {
		t.Run("Bool", func(t *testing.T) {
			a := glee.NewArray(0, 4)
			a = a.Store(glee.NewConstantExpr(3, 32), glee.NewConstantExpr(1, 1), false)
			if expr, ok := a.Select(glee.NewConstantExpr(3, 32), 1, false).(*glee.ConstantExpr); !ok {
				t.Fatal("expected constant expr")
			} else if expr.Value != 1 {
				t.Fatal("unexpected value")
			} else if expr.Width != 1 {
				t.Fatal("unexpected width")
			}
		})

		t.Run("BigEndian", func(t *testing.T) {
			a := glee.NewArray(0, 4)
			a = a.Store(glee.NewConstantExpr(0, 32), glee.NewConstantExpr(0xAABBCCDD, 32), false)
			if expr, ok := a.Select(glee.NewConstantExpr(0, 32), 32, false).(*glee.ConstantExpr); !ok {
				t.Fatal("expected constant expr")
			} else if expr.Value != 0xAABBCCDD {
				t.Fatal("unexpected value")
			}
		})

		t.Run("LittleEndian", func(t *testing.T) {
			a := glee.NewArray(0, 4)
			a = a.Store(glee.NewConstantExpr(0, 32), glee.NewConstantExpr(0xAABBCCDD, 32), true)
			if expr, ok := a.Select(glee.NewConstantExpr(0, 32), 32, true).(*glee.ConstantExpr); !ok {
				t.Fatal("expected constant expr")
			} else if expr.Value != 0xAABBCCDD {
				t.Fatal("unexpected value")
			}
		})
	})

	t.Run("Symbolic", func(t *testing.T) {
		t.Run("Empty", func(t *testing.T) {
			t.Run("SingleByte", func(t *testing.T) {
				a := glee.NewArray(0, 4)
				if diff := cmp.Diff(
					a.Select(glee.NewConstantExpr64(0), 8, false),
					&glee.SelectExpr{
						Array: a,
						Index: glee.NewConstantExpr64(0),
					},
				); diff != "" {
					t.Fatal(diff)
				}
			})

			t.Run("BigEndian", func(t *testing.T) {
				a := glee.NewArray(0, 4)
				if diff := cmp.Diff(
					a.Select(glee.NewConstantExpr64(2), 16, false),
					&glee.ConcatExpr{
						MSB: &glee.SelectExpr{
							Array: a,
							Index: glee.NewConstantExpr64(2),
						},
						LSB: &glee.SelectExpr{
							Array: a,
							Index: glee.NewConstantExpr64(3),
						},
					},
				); diff != "" {
					t.Fatal(diff)
				}
			})

			t.Run("LittleEndian", func(t *testing.T) {
				a := glee.NewArray(0, 4)
				if diff := cmp.Diff(
					a.Select(glee.NewConstantExpr64(2), 16, true),
					&glee.ConcatExpr{
						MSB: &glee.SelectExpr{
							Array: a,
							Index: glee.NewConstantExpr64(3),
						},
						LSB: &glee.SelectExpr{
							Array: a,
							Index: glee.NewConstantExpr64(2),
						},
					},
				); diff != "" {
					t.Fatal(diff)
				}
			})

			// Ensure stores using selects from other arrays return references
			// to that original array's expressions.
			t.Run("MultiArray", func(t *testing.T) {
				a, b := glee.NewArray(0, 4), glee.NewArray(0, 8)
				b = b.Store(
					glee.NewConstantExpr64(6),
					a.Select(glee.NewConstantExpr64(2), 16, false),
					false,
				)

				if diff := cmp.Diff(
					&glee.ConcatExpr{
						MSB: &glee.SelectExpr{
							Array: b,
							Index: glee.NewConstantExpr64(4),
						},
						LSB: &glee.ConcatExpr{
							MSB: &glee.SelectExpr{
								Array: b,
								Index: glee.NewConstantExpr64(5),
							},
							LSB: &glee.ConcatExpr{
								MSB: &glee.SelectExpr{
									Array: a,
									Index: glee.NewConstantExpr64(2),
								},
								LSB: &glee.SelectExpr{
									Array: a,
									Index: glee.NewConstantExpr64(3),
								},
							},
						},
					},
					b.Select(glee.NewConstantExpr64(4), 32, false),
				); diff != "" {
					t.Fatal(diff)
				}
			})

			// Ensure selection of an array that contains a store with a
			// symbolic index will simply a read from the array.
			t.Run("SymbolicIndex", func(t *testing.T) {
				a, b, c := glee.NewArray(0, 8), glee.NewArray(0, 8), glee.NewArray(0, 8)

				// Write concrete zeros.
				c = c.Store(
					glee.NewConstantExpr64(0),
					glee.NewConstantExpr64(0),
					false,
				)

				// Overwrite with store using symbolic index.
				c = c.Store(
					b.Select(glee.NewConstantExpr64(0), 32, false),
					a.Select(glee.NewConstantExpr64(0), 8, false),
					false,
				)

				if diff := cmp.Diff(
					&glee.ConcatExpr{
						MSB: &glee.SelectExpr{
							Array: c,
							Index: glee.NewConstantExpr64(0),
						},
						LSB: &glee.SelectExpr{
							Array: c,
							Index: glee.NewConstantExpr64(1),
						},
					},
					c.Select(glee.NewConstantExpr64(0), 16, false),
				); diff != "" {
					t.Fatal(diff)
				}
			})

			// Ensure that selection from an array with a symbolic store index
			// and then concrete store index will return the concrete store.
			t.Run("SymbolicIndexOverwritten", func(t *testing.T) {
				a, b, c := glee.NewArray(0, 4), glee.NewArray(0, 4), glee.NewArray(0, 4)
				c = c.Store(
					b.Select(glee.NewConstantExpr64(0), 32, false),
					a.Select(glee.NewConstantExpr64(0), 32, false),
					false,
				)

				c = c.Store(
					glee.NewConstantExpr64(1),
					a.Select(glee.NewConstantExpr64(0), 8, false),
					false,
				)

				if diff := cmp.Diff(
					&glee.ConcatExpr{
						MSB: &glee.SelectExpr{
							Array: c,
							Index: glee.NewConstantExpr64(0),
						},
						LSB: &glee.SelectExpr{
							Array: a,
							Index: glee.NewConstantExpr64(0),
						},
					},
					c.Select(glee.NewConstantExpr64(0), 16, false),
				); diff != "" {
					t.Fatal(diff)
				}
			})
		})
	})

	t.Run("GC", func(t *testing.T) {
		t.Run("ConcreteIndex", func(t *testing.T) {
			a := glee.NewArray(0, 2)
			a = a.Store(glee.NewConstantExpr64(0), glee.NewConstantExpr8(0), false)
			a = a.Store(glee.NewConstantExpr64(1), glee.NewConstantExpr8(1), false)
			a = a.Store(glee.NewConstantExpr64(0), glee.NewConstantExpr8(2), false)
			if expr, ok := a.Select(glee.NewConstantExpr64(0), 16, false).(*glee.ConstantExpr); !ok {
				t.Fatal("expected constant expr")
			} else if expr.Value != 0x0201 {
				t.Fatalf("unexpected value: 0x%04x", expr.Value)
			}

			if diff := cmp.Diff(
				&glee.Array{
					Size: 2,
					Updates: &glee.ArrayUpdate{
						Index: glee.NewConstantExpr64(0),
						Value: glee.NewConstantExpr8(2),
						Next: &glee.ArrayUpdate{
							Index: glee.NewConstantExpr64(1),
							Value: glee.NewConstantExpr8(1),
						},
					},
				},
				a,
			); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("SymbolicIndex", func(t *testing.T) {
			a, b := glee.NewArray(0, 2), glee.NewArray(0, 1)
			a = a.Store(glee.NewConstantExpr64(0), glee.NewConstantExpr8(0), false)
			a = a.Store(b.Select(glee.NewConstantExpr64(0), 8, false), glee.NewConstantExpr8(1), false) // symbolic index
			a = a.Store(glee.NewConstantExpr64(0), glee.NewConstantExpr8(2), false)

			if diff := cmp.Diff(
				&glee.Array{
					Size: 2,
					Updates: &glee.ArrayUpdate{
						Index: glee.NewConstantExpr64(0),
						Value: glee.NewConstantExpr8(2),
						Next: &glee.ArrayUpdate{
							Index: &glee.CastExpr{
								Src: &glee.SelectExpr{
									Array: b,
									Index: glee.NewConstantExpr64(0),
								},
								Width: 64,
							},
							Value: glee.NewConstantExpr8(1),
							Next: &glee.ArrayUpdate{
								Index: glee.NewConstantExpr64(0),
								Value: glee.NewConstantExpr8(0),
							},
						},
					},
				},
				a,
			); diff != "" {
				t.Fatal(diff)
			}
		})
	})

	t.Run("IsSymbolic", func(t *testing.T) {
		t.Run("AllConcrete", func(t *testing.T) {
			a := glee.NewArray(0, 2)
			a = a.Store(glee.NewConstantExpr(0, 32), glee.NewConstantExpr(0, 8), false)
			a = a.Store(glee.NewConstantExpr(1, 32), glee.NewConstantExpr(0, 8), false)
			if a.IsSymbolic() {
				t.Fatal("expected concrete")
			}
		})

		t.Run("UnsetByte", func(t *testing.T) {
			a := glee.NewArray(0, 2)
			a = a.Store(glee.NewConstantExpr(0, 32), glee.NewConstantExpr(0, 8), false)
			if !a.IsSymbolic() {
				t.Fatal("expected symbolic")
			}
		})

		t.Run("ContainsSelectValue", func(t *testing.T) {
			a, b := glee.NewArray(0, 2), glee.NewArray(0, 2)
			a = a.Store(glee.NewConstantExpr(0, 32), glee.NewConstantExpr(0, 8), false)
			a = a.Store(glee.NewConstantExpr(1, 32), b.Select(glee.NewConstantExpr(0, 32), 8, false), false)
			if !a.IsSymbolic() {
				t.Fatal("expected symbolic")
			}
		})

		t.Run("ContainsSelectIndex", func(t *testing.T) {
			a, b := glee.NewArray(0, 2), glee.NewArray(0, 2)
			a = a.Store(glee.NewConstantExpr(0, 32), glee.NewConstantExpr(0, 8), false)
			a = a.Store(b.Select(glee.NewConstantExpr(0, 32), 8, false), glee.NewConstantExpr(0, 32), false)
			if !a.IsSymbolic() {
				t.Fatal("expected symbolic")
			}
		})
	})
}

func TestCompareArray(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if cmp := glee.CompareArray(nil, nil); cmp != 0 {
			t.Fatalf("unexpected compare: %d", cmp)
		} else if cmp := glee.CompareArray(nil, glee.NewArray(0, 2)); cmp != -1 {
			t.Fatalf("unexpected compare: %d", cmp)
		} else if cmp := glee.CompareArray(glee.NewArray(0, 2), nil); cmp != 1 {
			t.Fatalf("unexpected compare: %d", cmp)
		}
	})

	t.Run("Size", func(t *testing.T) {
		if cmp := glee.CompareArray(glee.NewArray(0, 2), glee.NewArray(0, 2)); cmp != 0 {
			t.Fatalf("unexpected compare: %d", cmp)
		} else if cmp := glee.CompareArray(glee.NewArray(0, 1), glee.NewArray(0, 2)); cmp != -1 {
			t.Fatalf("unexpected compare: %d", cmp)
		} else if cmp := glee.CompareArray(glee.NewArray(0, 2), glee.NewArray(0, 1)); cmp != 1 {
			t.Fatalf("unexpected compare: %d", cmp)
		}
	})
}

func TestCompareArrayUpdate(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		upd := glee.NewArrayUpdate(glee.NewConstantExpr(0, 32), glee.NewConstantExpr(0, 8), nil)
		if cmp := glee.CompareArrayUpdate(nil, nil); cmp != 0 {
			t.Fatalf("unexpected compare: %d", cmp)
		} else if cmp := glee.CompareArrayUpdate(nil, upd); cmp != -1 {
			t.Fatalf("unexpected compare: %d", cmp)
		} else if cmp := glee.CompareArrayUpdate(upd, nil); cmp != 1 {
			t.Fatalf("unexpected compare: %d", cmp)
		}
	})

	t.Run("Index", func(t *testing.T) {
		a := glee.NewArrayUpdate(glee.NewConstantExpr(0, 32), glee.NewConstantExpr(0, 8), nil)
		b := glee.NewArrayUpdate(glee.NewConstantExpr(1, 32), glee.NewConstantExpr(0, 8), nil)
		if cmp := glee.CompareArrayUpdate(a, a); cmp != 0 {
			t.Fatalf("unexpected compare: %d", cmp)
		} else if cmp := glee.CompareArrayUpdate(a, b); cmp != -1 {
			t.Fatalf("unexpected compare: %d", cmp)
		} else if cmp := glee.CompareArrayUpdate(b, a); cmp != 1 {
			t.Fatalf("unexpected compare: %d", cmp)
		}
	})

	t.Run("Value", func(t *testing.T) {
		a := glee.NewArrayUpdate(glee.NewConstantExpr(0, 32), glee.NewConstantExpr(0, 8), nil)
		b := glee.NewArrayUpdate(glee.NewConstantExpr(0, 32), glee.NewConstantExpr(1, 8), nil)
		if cmp := glee.CompareArrayUpdate(a, a); cmp != 0 {
			t.Fatalf("unexpected compare: %d", cmp)
		} else if cmp := glee.CompareArrayUpdate(a, b); cmp != -1 {
			t.Fatalf("unexpected compare: %d", cmp)
		} else if cmp := glee.CompareArrayUpdate(b, a); cmp != 1 {
			t.Fatalf("unexpected compare: %d", cmp)
		}
	})

	t.Run("Next", func(t *testing.T) {
		a := glee.NewArrayUpdate(glee.NewConstantExpr(0, 32), glee.NewConstantExpr(0, 8), nil)
		b := glee.NewArrayUpdate(glee.NewConstantExpr(0, 32), glee.NewConstantExpr(0, 8), glee.NewArrayUpdate(glee.NewConstantExpr(0, 32), glee.NewConstantExpr(0, 8), nil))
		if cmp := glee.CompareArrayUpdate(a, a); cmp != 0 {
			t.Fatalf("unexpected compare: %d", cmp)
		} else if cmp := glee.CompareArrayUpdate(a, b); cmp != -1 {
			t.Fatalf("unexpected compare: %d", cmp)
		} else if cmp := glee.CompareArrayUpdate(b, a); cmp != 1 {
			t.Fatalf("unexpected compare: %d", cmp)
		}
	})
}
