package main

import (
	"errors"
	"log"
	"net"

	"fmt"

	"github.com/go-gl/mathgl/mgl32"
	pb "github.com/omustardo/scanner/protos/meshbuilder"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const port = ":50051"

type project struct {
	points []mgl32.Vec3
}

type Server struct {
	pb.MeshBuilderServer

	projects map[string]project
}

func (s *Server) CreateProject(ctx context.Context, req *pb.CreateProjectRequest) (*pb.CreateProjectResponse, error) {
	if _, ok := s.projects[req.Name]; ok {
		return nil, fmt.Errorf("project already exists with name %q", req.Name)
	}
	s.projects[req.Name] = project{}
	return &pb.CreateProjectResponse{}, nil
}

func (s *Server) Add(ctx context.Context, req *pb.AddRequest) (*pb.AddResponse, error) {
	if _, ok := s.projects[req.Name]; !ok {
		return nil, fmt.Errorf("unknown project: %q", req.Name)
	}
	project := s.projects[req.Name]
	project.points = append(project.points, processDepth(req.GetDepth())...)

	return &pb.AddResponse{}, nil
}

func processDepth(depth *pb.Depth) []mgl32.Vec3 {
	if validateDepth(depth) != nil {
		return nil
	}
	points := []mgl32.Vec3{}
	for row := range depth.Rows {
		if depth.Rows[row] == nil {
			continue
		}
		for col, value := range depth.Rows[row].Values {
			points = append(points, mgl32.Vec3{float32(row), float32(col), float32(value)})
		}
	}
	return points
}

func validateDepth(d *pb.Depth) error {
	if d == nil || d.Rows == nil {
		return nil
	}
	width := len(d.Rows[0].Values)
	for i := range d.Rows {
		if len(d.Rows[i].Values) != width {
			return fmt.Errorf("expected all rows in depth to be of equal size. got %v and %v", width, len(d.Rows[i].Values))
		}
	}
	return nil
}

func (s *Server) Retrieve(ctx context.Context, req *pb.RetrieveRequest) (*pb.RetrieveResponse, error) {
	return &pb.RetrieveResponse{}, errors.New("Retrieve is unimplemented")
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterMeshBuilderServer(s, &Server{})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
