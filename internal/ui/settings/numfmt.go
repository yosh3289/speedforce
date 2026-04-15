package settings

import "strconv"

func fmtInt(i int) string            { return strconv.Itoa(i) }
func parseInt(s string) (int, error) { return strconv.Atoi(s) }
