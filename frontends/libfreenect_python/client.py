#import the necessary modules
import freenect
import cv2
import numpy as np
import meshbuilder_pb2

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
  depth = get_depth()

  while (cv2.waitKey(10) != 32):
    #get a frame from depth sensor
    depth = get_depth()
    #display depth image
    cv2.imshow('stream',depth)

  first_point_cloud = depth
  cv2.imshow('first', first_point_cloud)

  while (cv2.waitKey(10) != 32):
    #get a frame from depth sensor
    depth = get_depth()
    #display depth image
    cv2.imshow('stream',depth)

  second_point_cloud = depth

  cv2.imshow('second', second_point_cloud)

  # quit program when 'esc' key is pressed
  while (cv2.waitKey(0) != 27):
    continue

  cv2.destroyAllWindows()

if __name__ == "__main__":
  main()
