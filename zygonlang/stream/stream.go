package stream

type Stream[T comparable] struct {
	Index    int
	Contents []T
}

func (stream *Stream[T]) peek(amount int) *T {
	index := stream.Index + amount

	if index >= len(stream.Contents) {
		return nil
	}
	return &stream.Contents[index]
}

func (stream *Stream[T]) consume(amount int) {
	stream.Index += amount
}
