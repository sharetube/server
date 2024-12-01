package randstr

import (
	"math/rand"
	"time"
)

const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

type Generator struct {
	rand  *rand.Rand
	bytes []byte
}

func New(chars []byte) *Generator {
	return &Generator{
		rand:  rand.New(rand.NewSource(time.Now().UnixNano())),
		bytes: chars,
	}
}

func (g Generator) GenerateRandomString(length int) string {
	b := make([]byte, length)
	for i, cache, remain := length-1, g.rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = g.rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(g.bytes) {
			b[i] = g.bytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return string(b)
}
