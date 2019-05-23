package main

func main() {
	var m = make([]int, 100)
	for i := 0; i < 100; i++ {
		go func() {
			m[i] = 1
		}()
	}
}
