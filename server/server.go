package server

import (
	"encoding/json"
	"errors"
	"github.com/heyujiang/hrpc/codec"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
)

const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int
	CodecType   codec.Type
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GodType,
}

type Server struct {
	serviceMap sync.Map
}

func NewServer() *Server {
	return &Server{}
}

var defaultServer = NewServer()

func (ser *Server) Register(rcvr interface{}) error {
	s := newService(rcvr)
	if _, dup := ser.serviceMap.LoadOrStore(s.name, s); dup {
		return errors.New("rpc : service already defined : " + s.name)
	}
	return nil
}

func Register(rcvr interface{}) error {
	return defaultServer.Register(rcvr)
}

func (ser *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server : accept error:", err)
			return
		}
		go ser.ServerConn(conn)
	}
}

func Accept(lis net.Listener) {
	defaultServer.Accept(lis)
}

func (ser *Server) findService(serviceMethod string) (svc *service, mtype *methodType, err error) {
	dot := strings.LastIndex(serviceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc server : service/method request ill-formed: " + serviceMethod)
		return
	}
	serviceName, methodName := serviceMethod[:dot], serviceMethod[dot+1:]
	scvi, ok := ser.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc server : can`t find service " + serviceName)
		return
	}
	svc = scvi.(*service)
	mtype = svc.method[methodName]
	if mtype == nil {
		err = errors.New("rpc server : can`t find method " + methodName)
	}
	return
}

func (ser *Server) ServerConn(conn io.ReadWriteCloser) {
	defer func() { _ = conn.Close() }()
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", opt.MagicNumber)
		return
	}
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("rpc server : invalid codec type %s", opt.CodecType)
	}
	ser.serveCodec(f(conn))
}

var invalidRequest = struct{}{}

func (ser *Server) serveCodec(cc codec.Codec) {
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	for {
		req, err := ser.readRequest(cc)
		if err != nil {
			if req == nil {
				break
			}
			req.h.Error = err.Error()
			ser.sendResponse(cc, req.h, invalidRequest, sending)
		}
		wg.Add(1)
		go ser.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()
	_ = cc.Close()
}

func (ser *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error: ", err)
		}
		return nil, err
	}
	return &h, nil
}

type request struct {
	h            *codec.Header
	argv, replyv reflect.Value
	mtype        *methodType
	svc          *service
}

func (ser *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := ser.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}

	req.svc, req.mtype, err = ser.findService(h.ServiceMethod)
	if err != nil {
		return req, err
	}
	req.argv = req.mtype.newArgv()
	req.replyv = req.mtype.newReplyv()

	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}
	if err = cc.ReadBody(argvi); err != nil {
		log.Println("rpc server : read body err:", err)
		return req, err
	}
	return req, err
}

func (ser *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response error: ", err)
	}
}

func (ser *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()

	err := req.svc.call(req.mtype, req.argv, req.replyv)
	if err != nil {
		req.h.Error = err.Error()
		ser.sendResponse(cc, req.h, invalidRequest, sending)
		return
	}
	ser.sendResponse(cc, req.h, req.replyv.Interface(), sending)
}
