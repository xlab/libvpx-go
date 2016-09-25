package main

func s(v string) string {
	return v + "\x00"
}

func b(v int32) bool {
	return v == 1
}
