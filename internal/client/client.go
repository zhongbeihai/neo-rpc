package client

// func (t *T) MethodName(argType T1, replyType *T2) error

/*
	Call represent an active RPC
*/
type Call struct{
	Seq uint64
	ServiceMethod string
	Args interface{}
	Reply interface{}
	Error error
	Done chan *Call
}

func (call *Call) done(){
	call.Done <- call
}

