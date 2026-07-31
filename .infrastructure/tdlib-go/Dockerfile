FROM golang:1.26.5-trixie

WORKDIR /td

RUN apt-get update -y
RUN apt-get upgrade -y
RUN apt-get install make git zlib1g-dev libssl-dev gperf php-cli cmake clang libc++-dev libc++abi-dev -y

RUN git clone https://github.com/tdlib/td.git .

RUN rm -rf build
RUN mkdir build

WORKDIR /td/build

RUN CXXFLAGS="-stdlib=libc++" CC=/usr/bin/clang CXX=/usr/bin/clang++ cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX:PATH=/usr/local ..

RUN cmake --build . --target install

RUN rm -rf /td

ENV CGO_LDFLAGS="-Wl,-rpath,/usr/local/lib -L/usr/local/lib -ltdjson -lc++"
ENV CGO_CFLAGS=-I/usr/local/include
