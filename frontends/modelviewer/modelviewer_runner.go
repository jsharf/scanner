package main

import (
	"flag"
	"image/color"
	"log"
	"math"
	"os"
	"time"

	"google.golang.org/grpc"

	"fmt"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/goxjs/gl"
	"github.com/goxjs/glfw"
	"github.com/omustardo/gome"
	"github.com/omustardo/gome/camera"
	"github.com/omustardo/gome/camera/zoom"
	"github.com/omustardo/gome/input/keyboard"
	"github.com/omustardo/gome/input/mouse"
	"github.com/omustardo/gome/model"
	"github.com/omustardo/gome/shader"
	"github.com/omustardo/gome/util/fps"
	"github.com/omustardo/gome/util/glutil"
	"github.com/omustardo/gome/view"
	"github.com/omustardo/scanner/protos/meshbuilder"
	"golang.org/x/net/context"
)

const (
	address     = "localhost:50051"
	meshProject = "testProject"
)

var (
	windowWidth  = flag.Int("window_width", 1000, "initial window width")
	windowHeight = flag.Int("window_height", 1000, "initial window height")

	frameRate = flag.Duration("framerate", time.Second/60, `Cap on framerate. Provide with units, like "16.66ms"`)
)

func init() {
	// log print with .go file and line number.
	log.SetFlags(log.Lshortfile)
	log.SetOutput(os.Stdout)
}

func main() {
	flag.Parse()

	client, conn, err := NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	terminate := gome.Initialize("Animation Demo", *windowWidth, *windowHeight, "")
	defer terminate()

	shader.Model.SetAmbientLight(&color.NRGBA{60, 60, 60, 0}) // 3D objects don't look 3D in the default max lighting, so tone it down.

	req := &meshbuilder.RetrieveRequest{Name: meshProject}
	resp, err := client.Retrieve(context.Background(), req)
	if err != nil {
		log.Fatal("Retrieve error:", err)
	}
	if resp == nil {
		log.Fatal("no response from client.Retrieve")
	}
	if len(resp.Points) == 0 {
		log.Fatal("no data from client.Retrieve")
	}
	points := toVec3(resp.Points)
	fmt.Println(len(points))
	vertexVBO := glutil.LoadBufferVec3(points)

	// Player is an empty model. It has no mesh so it can't be rendered, but it can still exist in the world.
	player := &model.Model{}
	player.Position[0] = 0
	cam := &camera.TargetCamera{
		Target:       player,
		TargetOffset: mgl32.Vec3{0, -1500, 1000},
		Up:           mgl32.Vec3{0, 1, 0},
		Zoomer: zoom.NewScrollZoom(0.1, 3,
			func() float32 {
				return mouse.Handler.Scroll().Y()
			},
		),
		Near: 0.1,
		Far:  10000,
		FOV:  math.Pi / 4.0,
	}

	ticker := time.NewTicker(*frameRate)
	for !view.Window.ShouldClose() {
		glfw.PollEvents() // Reads window events, like keyboard and mouse input.
		fps.Handler.Update()
		keyboard.Handler.Update()
		mouse.Handler.Update()

		ApplyInputs(player, cam)

		// Set up Model-View-Projection Matrix and send it to the shader program.
		mvMatrix := cam.ModelView()
		w, h := view.Window.GetSize()
		pMatrix := cam.ProjectionPerspective(float32(w), float32(h))
		shader.Model.SetMVPMatrix(pMatrix, mvMatrix)

		cam.Update()
		// Clear screen, then Draw everything
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		model.RenderXYZAxes()

		shader.Model.SetTranslationMatrix(0, 0, 0)
		shader.Model.SetRotationMatrixQ(mgl32.QuatIdent())
		shader.Model.SetScaleMatrix(1, 1, 1)
		shader.Model.SetColor(&color.NRGBA{255, 255, 255, 255})
		gl.BindBuffer(gl.ARRAY_BUFFER, vertexVBO)
		gl.EnableVertexAttribArray(shader.Model.VertexPositionAttrib) // TODO: Can these VertexAttribArrays be enabled a single time in shader initialization and then just always used?
		gl.VertexAttribPointer(shader.Model.VertexPositionAttrib, 3, gl.FLOAT, false, 0, 0)
		gl.DrawArrays(gl.POINTS, 0, len(points))

		// Swaps the buffer that was drawn on to be visible. The visible buffer becomes the one that gets drawn on until it's swapped again.
		view.Window.SwapBuffers()
		<-ticker.C // wait up to the framerate cap.
	}
}

func ApplyInputs(target *model.Model, cam camera.Camera) {
	var move mgl32.Vec2
	if keyboard.Handler.IsKeyDown(glfw.KeyA, glfw.KeyLeft) {
		move[0] += -1
	}
	if keyboard.Handler.IsKeyDown(glfw.KeyD, glfw.KeyRight) {
		move[0] += 1
	}
	if keyboard.Handler.IsKeyDown(glfw.KeyW, glfw.KeyUp) {
		move[1] += 1
	}
	if keyboard.Handler.IsKeyDown(glfw.KeyS, glfw.KeyDown) {
		move[1] += -1
	}
	moveSpeed := float32(500)
	move = move.Normalize().Mul(moveSpeed * fps.Handler.DeltaTimeSeconds())
	target.ModifyPosition(move[0], move[1], 0)
}

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
