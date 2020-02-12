package service

import (
	"context"
	"log"
	"net"
	"os"
	"src/chat_service/proto"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	glog "google.golang.org/grpc/grpclog"
)

var grpcLog glog.LoggerV2
var wg sync.WaitGroup

func init() {
	grpcLog = glog.NewLoggerV2(os.Stdout, os.Stdout, os.Stdout)

}

type Connection struct {
	stream proto.Messinger_CreateStreamServer
	id     string
	active bool
	error  chan error
}
type Server struct {
	Connections map[string]*Connection
	Users       map[string]*proto.User
}

func initServer() *Server {
	return &Server{
		Connections: make(map[string]*Connection),
		Users:       make(map[string]*proto.User),
	}

}
func (s *Server) Signup(ctx context.Context, request *proto.SignupRequest) (*proto.SignupResponse, error) {
	_, exist := s.Users[request.GetUser().GetId()]

	if exist {
		grpcLog.Errorf("Could not sign up user already exist")
		return &proto.SignupResponse{
			Done: false,
			Msg:  "User name already exists",
		}, nil
	}
	s.Users[request.GetUser().GetId()] = request.GetUser()
	grpcLog.Info("Succesful signup: ", request.GetUser().GetId())
	return &proto.SignupResponse{
		Done: true,
		Msg:  "",
	}, nil

}
func (s *Server) AddFriend(ctx context.Context, request *proto.AddFriendRequest) (*proto.AddFriendResponse, error) {

	id := request.GetId()
	user, exist := s.Users[id]
	if exist {
		grpcLog.Info("Add Contact request success")
		return &proto.AddFriendResponse{
			Done: true,
			User: user,
			Msg:  "",
		}, nil

	}
	grpcLog.Error("Add contact request faild")
	return &proto.AddFriendResponse{
		Done: false,
		User: nil,
		Msg:  "User does not exist",
	}, nil

}
func (s *Server) UploadMedia(stream proto.Messinger_UploadMediaServer) error {
	/*	var file []byte
		var info []byte

		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {

				err = stream.SendAndClose(&proto.MediaResponse{Done: true, Error: ""})
				return nil

			}
			err = errors.Wrapf(err, "failed to upload")
			return err

		}
		if msg.Media.GetFile != nil {
			info = msg.Media.GetFile()

		}
		if msg.Media.GetImage != nil {
			info = msg.GetMedia().GetImage()

		}
		if msg.Media.GetVideo != nil {
			info = msg.GetMedia().GetVideo()

		}
		file = append(file, info...)
		_, err = s.SendMessage(context.Background(), msg)

		return err*/
	return nil

}

func (s *Server) CreateStream(request *proto.RegisterRequest, stream proto.Messinger_CreateStreamServer) error {
	conn := &Connection{
		stream: stream,
		id:     request.User.Id,
		active: true,
		error:  make(chan error),
	}
	s.Connections[conn.id] = conn
	return <-conn.error
}
func (s *Server) SendMessage(ctx context.Context, msg *proto.Message) (*proto.Close, error) {
	wg := sync.WaitGroup{}
	conn, exist := s.Connections[msg.TargetId]
	if exist {
		wg.Add(1)
		go func(msg *proto.Message, conn *Connection) {
			defer wg.Done()
			if conn.active {
				err := conn.stream.Send(msg)
				grpcLog.Info("Sending message to: ", msg.TargetId)
				if err != nil {
					grpcLog.Errorf("Error with  Stream: %v - Error:%v", conn.stream, err)
					conn.active = false
					conn.error <- err

				}
			}

		}(msg, conn)
		wg.Wait()

	}

	return &proto.Close{}, nil
}

func Run() {
	server := initServer()
	creds, err := credentials.NewServerTLSFromFile("chat.crt", "chat.key")
	if err != nil {
		log.Fatal("cant find the certificate")
	}
	grpcServer := grpc.NewServer(grpc.Creds(creds))
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("error creating server: %v", err)

	}
	grpcLog.Info("Starting server at port: 8080")
	proto.RegisterMessingerServer(grpcServer, server)
	grpcServer.Serve(listener)

}
