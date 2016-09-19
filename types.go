package depman

type JsonAble interface {
	ToJsonString() (string, error)
	ToString() string
}
