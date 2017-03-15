package main

import (
	"fmt"
	"log"
	"net"

	pb "github.com/omustardo/scanner/protos/meshbuilder"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const port = ":50051"

type project struct {
	points []*pb.Point
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
	log.Println("Created project:", req.Name)
	return &pb.CreateProjectResponse{}, nil
}

func (s *Server) Add(ctx context.Context, req *pb.AddRequest) (*pb.AddResponse, error) {
	if _, ok := s.projects[req.Name]; !ok {
		return nil, fmt.Errorf("unknown project: %q", req.Name)
	}
	log.Println("Add request for", len(req.Depth.Rows), "rows")
	project := s.projects[req.Name]
	project.points = append(project.points, processDepth(req.GetDepth())...)
	s.projects[req.Name] = project

	log.Println("Added stuff. Project", req.Name, "has", len(project.points), " points.")
	return &pb.AddResponse{}, nil
}

func processDepth(depth *pb.Depth) []*pb.Point {
	if validateDepth(depth) != nil {
		return nil
	}
	points := []*pb.Point{}
	for row := range depth.Rows {
		if depth.Rows[row] == nil {
			continue
		}
		for col, value := range depth.Rows[row].Values {
			points = append(points, &pb.Point{X: float32(row), Y: float32(col), Z: float32(value)})
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
	if _, ok := s.projects[req.Name]; !ok {
		return nil, fmt.Errorf("unknown project: %q", req.Name)
	}
	log.Println("Retrieving from project", req.Name)
	log.Println(len(s.projects[req.Name].points), "values")
	if len(s.projects[req.Name].points) > 3 {
		log.Println(s.projects[req.Name].points[:3])
	}
	return &pb.RetrieveResponse{Points: s.projects[req.Name].points}, nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	meshBuilder := &Server{}
	meshBuilder.projects = make(map[string]project)
	meshBuilder.projects["test"] = project{points: []*pb.Point{{X: 10, Y: 10, Z: 10}}}
	s := grpc.NewServer()
	pb.RegisterMeshBuilderServer(s, meshBuilder)
	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
