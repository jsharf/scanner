#import the necessary modules
import freenect
import cv2
import numpy as np
import meshbuilder_pb2
import meshbuilder_pb2_grpc
import grpc
import sys

#function to get RGB image from kinect
def get_video():
  array,_ = freenect.sync_get_video()
  array = cv2.cvtColor(array,cv2.COLOR_RGB2BGR)
  return array

#function to get depth image from kinect
def get_depth():
  array,_ = freenect.sync_get_depth()
  array = array.astype(np.uint8)
  return array

def display_depth(dev, data, timestamp):
    global keep_running
    global depth
    depth = data.astype(np.uint8)
    cv2.imshow('Depth', depth)
    if cv2.waitKey(10) == 27:
        keep_running = False
  
frames = []
keep_running = True
depth = np.array([1])
last_sent_depth = np.array([1])
project_name = ""
stub = None


def loop(*args):
  global stub
  global depth
  global last_sent_depth
  global keep_running
  global frames
  global project_name

  if not keep_running:
    raise freenect.Kill

  if ((depth == last_sent_depth).all()):
    return

  print("New depth value!")

  # Stuff frame in proto.
  proto = meshbuilder_pb2.AddRequest()
  proto.name = project_name
  for row in depth:
    new_row = proto.depth.rows.add()
    new_row.values[:] = row
  proto.depth.x_fov = 58.5
  proto.depth.y_fov = 46.6
  
  print("depth_in_proto")

  # Start GRPC
  frames.append(stub.Add.future(proto))
  last_sent_depth = depth

  print("proto_sent_to_grpc")

  # Reap a frame. Don't spent time going through all of them...
  for i,_ in enumerate(frames):
    if frames[i].done():
      if (frames[i].exception() != None):
        print("Add grpc returned exception: {0}".format(str(frames[i].exception())))
      del frames[i]
      break
  
  print("rpc_frames_handled")

def main():
  global frames
  global project_name
  global stub
  if (len(sys.argv) != 3):
    print("Usage: python client.py address:port project_name")
    sys.exit(1)

  server = sys.argv[1]
  project_name = sys.argv[2]

  channel = grpc.insecure_channel(server)
  stub = meshbuilder_pb2_grpc.MeshBuilderStub(channel)

  print("Creating Project.")

  # Create the project (if it doesn't already exist) and get the project ID.
  create_project_proto = meshbuilder_pb2.CreateProjectRequest()
  create_project_proto.name = project_name
  stub.CreateProject(create_project_proto)
  
  print("Project created!.")


  print("Streaming frames...")

  print('Press ESC in window to stop')
  freenect.runloop(depth=display_depth,
                   video=None,
                   body=loop)

  cv2.destroyAllWindows()

if __name__ == "__main__":
  main()
