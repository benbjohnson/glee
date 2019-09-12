# Name: command-line-arguments.main
# Package: command-line-arguments
# Location: /Users/benbjohnson/src/glee/etc/ssa/main.go:9:6
func main():
0:                                                                entry P:0 S:0
	*x = 2:int
	t0 = *sl                                                          []int
	t1 = &t0[2:int]                                                    *int
	*t1 = 10:int
	t2 = &st.A [#0]                                                    *int
	t3 = *t2                                                            int
	t4 = *x                                                             int
	t5 = &st.B [#1]                                                 *string
	t6 = *t5                                                         string
	t7 = *sl                                                          []int
	t8 = new [4]interface{} (varargs)                       *[4]interface{}
	t9 = &t8[0:int]                                            *interface{}
	t10 = make interface{} <- int (t3)                          interface{}
	*t9 = t10
	t11 = &t8[1:int]                                           *interface{}
	t12 = make interface{} <- int (t4)                          interface{}
	*t11 = t12
	t13 = &t8[2:int]                                           *interface{}
	t14 = make interface{} <- string (t6)                       interface{}
	*t13 = t14
	t15 = &t8[3:int]                                           *interface{}
	t16 = make interface{} <- []int (t7)                        interface{}
	*t15 = t16
	t17 = slice t8[:]                                         []interface{}
	t18 = fmt.Printf("%d %d %s %v\n":string, t17...)     (n int, err error)
	return

# Name: command-line-arguments.init
# Package: command-line-arguments
# Synthetic: package initializer
func init():
0:                                                                entry P:0 S:2
	t0 = *init$guard                                                   bool
	if t0 goto 2 else 1
1:                                                           init.start P:1 S:1
	*init$guard = true:bool
	t1 = fmt.init()                                                      ()
	*x = 1:int
	t2 = &st.A [#0]                                                    *int
	t3 = *x                                                             int
	t4 = &st.B [#1]                                                 *string
	*t2 = t3
	*t4 = "foo":string
	t5 = new [3]int (slicelit)                                      *[3]int
	t6 = &t5[0:int]                                                    *int
	*t6 = 1:int
	t7 = &t5[1:int]                                                    *int
	*t7 = 2:int
	t8 = &t5[2:int]                                                    *int
	*t8 = 3:int
	t9 = slice t5[:]                                                  []int
	*sl = t9
	jump 2
2:                                                            init.done P:2 S:0
	return

