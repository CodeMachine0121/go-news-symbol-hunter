package interfaces

type IRandomProxy interface {
	GenerateBytes(length int) []byte
}
