# Get required python library with pip install grpcio-tools.
python -m grpc_tools.protoc -I./ --python_out=. --grpc_python_out=. ./meshbuilder.proto
