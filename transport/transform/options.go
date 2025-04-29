package transform

type Option func(*options)

type options struct {
	transformer   TransformFunc
	detransformer TransformFunc
	maxPacketSize int
	batchSize     int
}

func getDefaultOptions() *options {
	return &options{
		maxPacketSize: _maxPacketSize,
	}
}

func WithTransformer(t TransformFunc) Option {
	return func(o *options) {
		o.transformer = t
	}
}

func WithDetransformer(t TransformFunc) Option {
	return func(o *options) {
		o.detransformer = t
	}
}

func WithMaxPacketSize(t int) Option {
	return func(o *options) {
		o.maxPacketSize = t
	}
}

func WithBatchSize(t int) Option {
	return func(o *options) {
		o.batchSize = t
	}
}
