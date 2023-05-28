package pkg

func ToPtr[V any](v V) *V {
	return &v
}
