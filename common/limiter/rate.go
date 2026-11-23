package limiter

import (
	"context"
	"io"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/buf"
	"golang.org/x/time/rate"
)

type Writer struct {
	writer  buf.Writer
	limiter *rate.Limiter
	w       io.Writer
}

func (l *Limiter) RateWriter(writer buf.Writer, limiter *rate.Limiter) buf.Writer {
	return &Writer{
		writer:  writer,
		limiter: limiter,
	}
}

func (w *Writer) Close() error {
	return common.Close(w.writer)
}

func (w *Writer) WriteMultiBuffer(mb buf.MultiBuffer) error {
	ctx := context.Background()
	if err := waitBytes(ctx, w.limiter, int(mb.Len())); err != nil {
		buf.ReleaseMulti(mb)
		return err
	}
	return w.writer.WriteMultiBuffer(mb)
}

// Wait in burst-sized chunks: WaitN rejects a whole buffer larger than Burst.
func waitBytes(ctx context.Context, bucket *rate.Limiter, n int) error {
	for n > 0 {
		step := bucket.Burst()
		if step < 1 {
			step = 1
		}
		if n < step {
			step = n
		}
		if err := bucket.WaitN(ctx, step); err != nil {
			return err
		}
		n -= step
	}
	return nil
}

type Reader struct {
	reader  buf.Reader
	limiter *rate.Limiter
	ctx     context.Context
}

func (l *Limiter) RateReader(ctx context.Context, reader buf.Reader, bucket *rate.Limiter) buf.Reader {
	return &Reader{reader: reader, limiter: bucket, ctx: ctx}
}
func (r *Reader) ReadMultiBuffer() (buf.MultiBuffer, error) {
	mb, err := r.reader.ReadMultiBuffer()
	if waitErr := waitBytes(r.ctx, r.limiter, int(mb.Len())); waitErr != nil {
		buf.ReleaseMulti(mb)
		return nil, waitErr
	}
	return mb, err
}
func (r *Reader) Interrupt() { common.Interrupt(r.reader) }
