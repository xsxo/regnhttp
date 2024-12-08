package regn

import (
	"bufio"
	"bytes"
	"testing"
)

func TestBytesPool(T *testing.T) {
	buffer := bytes.Buffer{}
	buffer.WriteString(Name + " - " + Version)

	pool := bufferPool.Get()

	pool.ReadFrom(&buffer)

	if pool.Len() == 0 {
		T.Error("Error use bytes pool buffer")
	}

	pool.Reset()
	buffer.Reset()

	pool.WriteString(Description)

	if pool.Len() == 0 {
		T.Error("Error use bytes pool buffer")
	}

	pool.Reset()
}

func TestWriterPool(T *testing.T) {
	buffer := bytes.Buffer{}

	flusherPool.Put(bufio.NewWriter(&buffer))

	writer := flusherPool.Get().(*bufio.Writer)
	writer.Reset(&buffer)

	if Writed, NewErr := writer.WriteString(Name + " - " + Version); NewErr != nil || Writed == 0 {
		T.Error("Error use Writer Pool -> WriteString function")
	}

	if NewErr := writer.Flush(); NewErr != nil {
		T.Error("Error use Writer Pool -> Flush function")
	}

	buffer.Reset()
}

func TestReaderPool(T *testing.T) {
	var buffer *bytes.Buffer = &bytes.Buffer{}
	peekerPool.Put(bufio.NewReader(buffer))

	Reader := peekerPool.Get().(*bufio.Reader)
	Reader.Reset(buffer)

	buffer.WriteString(Name + " - " + Version)

	Reader.Peek(1)
	buffered := Reader.Buffered()

	if buffered == 0 {
		T.Error("Error use Reader Pool -> Buffered function")
	}

	Readed, NewErr := Reader.Peek(buffered)
	if NewErr != nil || len(Readed) != buffered {
		T.Error("Error use Reader Pool -> Peek function")
	}

	Discarded, NewErr := Reader.Discard(len(Readed))
	if NewErr != nil || Discarded != buffered || buffer.Len() != 0 {
		T.Error("Error use Reader Pool -> Discard function")
	}

	buffer.Reset()
}
