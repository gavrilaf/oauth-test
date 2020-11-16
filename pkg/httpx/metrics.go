package httpx

//go:generate mockery --name Metrics --outpkg httpxmock --output ./httpxmock --dir .
type Metrics interface {
	RecordCount(key string)
}


