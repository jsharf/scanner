package main

import (
	"flag"
	"image/color"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/golang/protobuf/proto"
	"github.com/gonum/matrix/mat64"
	"github.com/goxjs/gl"
	"github.com/goxjs/glfw"
	"github.com/omustardo/gome"
	"github.com/omustardo/gome/asset"
	"github.com/omustardo/gome/camera"
	"github.com/omustardo/gome/core/entity"
	"github.com/omustardo/gome/input/keyboard"
	"github.com/omustardo/gome/input/mouse"
	"github.com/omustardo/gome/model"
	"github.com/omustardo/gome/model/mesh"
	"github.com/omustardo/gome/shader"
	"github.com/omustardo/gome/util"
	"github.com/omustardo/gome/util/fps"
	"github.com/omustardo/gome/util/glutil"
	"github.com/omustardo/gome/view"
	"github.com/omustardo/scanner/algorithms"
	"github.com/omustardo/scanner/protos/meshbuilder"
)

const (
	address     = "localhost:50051"
	meshProject = "testProject"
)

var (
	windowWidth  = flag.Int("window_width", 1000, "initial window width")
	windowHeight = flag.Int("window_height", 1000, "initial window height")

	frameRate = flag.Duration("framerate", time.Second/60, `Cap on framerate. Provide with units, like "16.66ms"`)
	baseDir   = flag.String("base_dir", `C:\workspace\Go\src\github.com\omustardo\scanner\frontends\modelviewer`, "All file paths should be specified relative to this root.")
)

func init() {
	// log print with .go file and line number.
	log.SetFlags(log.Lshortfile)
	log.SetOutput(os.Stdout)
}

func main() {
	flag.Parse()

	terminate := gome.Initialize("Animation Demo", *windowWidth, *windowHeight, *baseDir)
	defer terminate()

	// shader.Model.SetAmbientLight(&color.NRGBA{60, 60, 60, 0}) // 3D objects don't look 3D in the default max lighting, so tone it down.
	shader.Model.SetAmbientLight(&color.NRGBA{255, 255, 255, 0})

	// =========== Read points from Server ===========
	//client, conn, err := NewClient()
	//if err != nil {
	//	log.Fatal(err)
	//}
	//defer conn.Close()
	//req := &meshbuilder.RetrieveRequest{Name: meshProject}
	//resp, err := client.Retrieve(context.Background(), req)
	//if err != nil {
	//	log.Fatal("Retrieve error:", err)
	//}
	//if resp == nil {
	//	log.Fatal("no response from client.Retrieve")
	//}
	//if len(resp.Points) == 0 {
	//	log.Fatal("no data from client.Retrieve")
	//}
	//points := toVec3(resp.Points)
	//fmt.Println(len(points))

	// =========== Read points from File ===========
	pointCloud := toVec3(fromFile(`1489724366`)) // 1489724360 1489724366
	log.Printf("got %d points, storing in texture of size %d\n", len(pointCloud), util.RoundUpToPowerOfTwo(len(pointCloud)))
	p := &points.PointCloudAnalyzer{}
	p.MakePointCloudAnalyzer(cloudToDense(pointCloud))
	texData := make([][]uint8, 0, util.RoundUpToPowerOfTwo(len(pointCloud)))
	texCoords := make([]mgl32.Vec2, 0, util.RoundUpToPowerOfTwo(len(pointCloud)))
	for i := range pointCloud {
		desc := p.Descriptor(i)
		c := desc.VisualizeDescriptor()
		texData = append(texData, []uint8{c.R, c.G, c.B, c.A})
		texCoords = append(texCoords, mgl32.Vec2{float32(i), 0})
	}
	tex, err := asset.LoadTextureData2D(texData)
	if err != nil {
		log.Fatal(err)
	}
	vertexVBO := glutil.LoadBufferVec3(pointCloud)
	m := model.Model{
		Mesh: mesh.NewMesh(
			vertexVBO,
			gl.Buffer{}, gl.Buffer{},
			gl.POINTS,
			len(pointCloud),
			&color.NRGBA{255, 255, 255, 255},
			tex,
			glutil.LoadBufferVec2(texCoords),
		),
		Entity: entity.Default(),
	}

	// Player is an empty model. It has no mesh so it can't be rendered, but it can still exist in the world.
	//player := &model.Model{}
	//player.Position[0] = 0
	cam := camera.NewFreeCamera()

	ticker := time.NewTicker(*frameRate)
	for !view.Window.ShouldClose() {
		glfw.PollEvents() // Reads window events, like keyboard and mouse input.
		fps.Handler.Update()
		keyboard.Handler.Update()
		mouse.Handler.Update()

		//ApplyInputs(player)

		// Set up Model-View-Projection Matrix and send it to the shader program.
		mvMatrix := cam.ModelView()
		w, h := view.Window.GetSize()
		pMatrix := cam.ProjectionPerspective(float32(w), float32(h)) // ProjectionOrthographic(float32(w), float32(h))
		shader.Model.SetMVPMatrix(pMatrix, mvMatrix)

		cam.Update(fps.Handler.DeltaTime())
		// Clear screen, then Draw everything
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		model.RenderXYZAxes()

		m.Render()

		// Swaps the buffer that was drawn on to be visible. The visible buffer becomes the one that gets drawn on until it's swapped again.
		view.Window.SwapBuffers()
		<-ticker.C // wait up to the framerate cap.
	}
}

//
//func ApplyInputs(target *model.Model) {
//	var move mgl32.Vec2
//	if keyboard.Handler.IsKeyDown(glfw.KeyA, glfw.KeyLeft) {
//		move[0] += -1
//	}
//	if keyboard.Handler.IsKeyDown(glfw.KeyD, glfw.KeyRight) {
//		move[0] += 1
//	}
//	if keyboard.Handler.IsKeyDown(glfw.KeyW, glfw.KeyUp) {
//		move[1] += 1
//	}
//	if keyboard.Handler.IsKeyDown(glfw.KeyS, glfw.KeyDown) {
//		move[1] += -1
//	}
//	moveSpeed := float32(500)
//	move = move.Normalize().Mul(moveSpeed * fps.Handler.DeltaTimeSeconds())
//	target.ModifyPosition(move[0], move[1], 0)
//}

func NewClient() (meshbuilder.MeshBuilderClient, *grpc.ClientConn, error) {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, nil, err
	}
	return meshbuilder.NewMeshBuilderClient(conn), conn, nil
}

func toVec3(p []*meshbuilder.Point) []mgl32.Vec3 {
	v := make([]mgl32.Vec3, len(p))
	for i := range p {
		v[i] = mgl32.Vec3{p[i].X, p[i].Y, p[i].Z}
	}
	return v
}

func fromFile(path string) []*meshbuilder.Point {
	data, err := asset.LoadFile(path)
	if err != nil {
		panic(err)
	}
	depth := &meshbuilder.Depth{}
	err = proto.Unmarshal(data, depth)
	if err != nil {
		panic(err)
	}
	log.Println("read from file:", path)
	return processDepth(depth)
}

func processDepth(depth *meshbuilder.Depth) []*meshbuilder.Point {
	p := []*meshbuilder.Point{}
	for row := range depth.Rows {
		if depth.Rows[row] == nil {
			continue
		}
		for col, value := range depth.Rows[row].Values {
			p = append(p, &meshbuilder.Point{X: float32(row), Y: float32(col), Z: float32(value)})
		}
	}
	if len(p) == 0 {
		panic("foo")
	}
	return p
}

func cloudToDense(vecs []mgl32.Vec3) *mat64.Dense {
	data := make([]float64, 0, len(vecs))
	for _, v := range vecs {
		data = append(data, float64(v.X()), float64(v.Y()), float64(v.Z()))
	}
	return mat64.NewDense(3, 640*480, data)
}
