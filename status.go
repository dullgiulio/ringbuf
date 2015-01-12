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

type Data struct {
	data   interface{}
	status ringbufStatus
}

func newData(status ringbufStatus, data interface{}) Data {
	return Data{data: data, status: status}
}
