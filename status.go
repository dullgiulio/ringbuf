package ringbuf

const (
	ringbufStatusUnknown = iota
	ringbufStatusOK
	ringbufStatusEOF
	ringbufStatusStarving
	ringbufStatusReader
	ringbufStatusReaderRequestCancel
	ringbufStatusReaderCancel
	ringbufStatusWrite
	ringbufStatusWriteOrStarve
)

type ringbufStatus int

type RingbufData struct {
	data   interface{}
	status ringbufStatus
}

func newRingbufData(status ringbufStatus, data interface{}) RingbufData {
	return RingbufData{data: data, status: status}
}
