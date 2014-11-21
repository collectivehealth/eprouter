package eprouter

const (
	internal_BUILD_NUMBER   = 49
	internal_VERSION_STRING = "0.9.1"
)

func BuildNumber() int64 {
	return internal_BUILD_NUMBER
}
func Version() string {
	return internal_VERSION_STRING
}
