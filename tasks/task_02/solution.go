package main

func rotateRunes(s string, shift int) string {
	curLen := len([]rune(s))

	if curLen == 0 {
		return ""
	}

	if shift == 0 {
		return string([]rune(s))
	}

	shift = -shift

	if shift > 0 {
		shift = shift % curLen
	} else {
		shift = curLen - (-shift % curLen)
	}

	runes := make([]rune, curLen)

	for i, sym := range []rune(s) {
		runes[(i+shift)%curLen] = sym
	}

	return string(runes)
}
