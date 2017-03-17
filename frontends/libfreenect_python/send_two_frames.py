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

def main():
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

  depth = get_depth()

  frames = []

  while (cv2.waitKey(10) != 32):
    #get a frame from depth sensor
    depth = get_depth()
    #display depth image
    cv2.imshow('stream',depth)

  first_point_cloud = depth
  
  # Stuff frame in proto.
  proto = meshbuilder_pb2.AddRequest()
  proto.name = project_name
  for row in first_point_cloud:
    new_row = proto.depth.rows.add()
    new_row.values[:] = row
  proto.depth.x_fov = 58.5
  proto.depth.y_fov = 46.6
  
  # Start GRPC
  frames.append(stub.Add.future(proto))

  while (cv2.waitKey(10) != 32):
    #get a frame from depth sensor
    depth = get_depth()
    #display depth image
    cv2.imshow('stream',depth)

  second_point_cloud = depth
  
  # Stuff frame in proto.
  proto = meshbuilder_pb2.AddRequest()
  proto.name = project_name
  for row in second_point_cloud:
    new_row = proto.depth.rows.add()
    new_row.values[:] = row
  proto.depth.x_fov = 58.5
  proto.depth.y_fov = 46.6

  # Start GRPC
  frames.append(stub.Add.future(proto))

  print("Sending frames... please wait.")

  # Reap frames.
  while (len(frames) > 0): 
    for i,_ in enumerate(frames):
      if frames[i].done():
        if (frames[i].exception() != None):
          print("Add grpc returned exception: {0}".format(str(frames[i].exception())))
        del frames[i]

  print("Frames sent!")

  # quit program when 'esc' key is pressed
  while (cv2.waitKey(0) != 27):
    continue

  cv2.destroyAllWindows()

if __name__ == "__main__":
  main()
