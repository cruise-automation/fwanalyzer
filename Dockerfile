FROM golang:1.13

RUN apt update && apt -y install e2tools mtools file squashfs-tools unzip python-setuptools python-lzo cpio sudo
RUN wget https://github.com/crmulliner/ubi_reader/archive/master.zip -O ubireader.zip && unzip ubireader.zip && cd ubi_reader-master && python setup.py install

WORKDIR $GOPATH/src/github.com/cruise-automation/fwanalyzer

COPY . ./

RUN make deps
