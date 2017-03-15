# Get required python library with pip install grpcio-tools.
python -m grpc_tools.protoc -I../../protos/meshbuilder/ --python_out=. --grpc_python_out=. ../../protos/meshbuilder/meshbuilder.proto
