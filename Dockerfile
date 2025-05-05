# # 第一阶段：构建Go二进制文件
# FROM golang:1.23.4 AS go-builder

# WORKDIR /Online3D
# COPY go.mod go.sum ./
# RUN go mod tidy

# COPY . .
# RUN GOPROXY=https://goproxy.cn,direct CGO_ENABLED=0 GOOS=linux go build -o main .

# # 第二阶段：构建Python环境
# FROM nvidia/cuda:11.8.0-devel-ubuntu20.04 AS python-builder

# # 安装基础工具
# RUN apt-get update && \
#     apt-get install -y --no-install-recommends \
#     nvidia-cuda-toolkit \
#     cmake \
#     ninja-build \
#     build-essential gfortran \
#     libpthread-stubs0-dev \
#     libgl1-mesa-glx libx11-6 libxext6 libsm6 libxrender1 \
#     libjpeg-dev libtiff-dev libopenexr-dev libpng-dev libwebp-dev \
#     libopenblas-dev \
#     liblapack-dev \
#     libjpeg-dev \
#     zlib1g-dev \
#     wget \
#     git \
#     build-essential \
#     ca-certificates && \
#     rm -rf /var/lib/apt/lists/*

# # 安装Miniconda
# ENV CONDA_DIR=/opt/conda
# ENV PATH=/opt/conda/bin:$PATH
# ENV CUDA_HOME=/user/local/cuda-11.8
# RUN wget --tries=5 --retry-connrefused --waitretry=30 \
#     https://mirrors.tuna.tsinghua.edu.cn/anaconda/miniconda/Miniconda3-latest-Linux-x86_64.sh -O ~/miniconda.sh && \
#     /bin/bash ~/miniconda.sh -b -p $CONDA_DIR && \
#     rm ~/miniconda.sh

# # 创建Conda环境
# COPY environment.yml .
# COPY submodules/ ./submodules/
# RUN $CONDA_DIR/bin/conda config --set show_channel_urls yes && \
#     $CONDA_DIR/bin/conda config --remove-key channels && \
#     $CONDA_DIR/bin/conda config --add channels https://mirrors.tuna.tsinghua.edu.cn/anaconda/cloud/conda-forge/ && \
#     $CONDA_DIR/bin/conda config --add channels https://mirrors.tuna.tsinghua.edu.cn/anaconda/pkgs/main/ && \
#     $CONDA_DIR/bin/conda config --add channels https://mirrors.tuna.tsinghua.edu.cn/anaconda/cloud/pytorch/ && \
#     $CONDA_DIR/bin/conda config --set remote_connect_timeout_secs 60 && \
#     $CONDA_DIR/bin/conda config --set remote_read_timeout_secs 1800 && \
#     $CONDA_DIR/bin/conda config --set remote_max_retries 10 && \
#     $CONDA_DIR/bin/conda install -y -n base conda-libmamba-solver && \
#     CONDA_SOLVER=libmamba $CONDA_DIR/bin/conda env create -n gaussian_splatting -f environment.yml --quiet && \
#     CONDA_SOLVER=libmamba $CONDA_DIR/bin/conda env update -n gaussian_splatting -f environment.yml && \
#     conda clean -afy

# RUN conda run -n gaussian_splatting pip install --no-cache-dir \
#     ./submodules/diff-gaussian-rasterization \
#     ./submodules/simple-knn \
#     ./submodules/fused-ssim
# # 第三阶段：最终镜像
# FROM nvidia/cuda:11.8.0-runtime-ubuntu20.04

# # 从Python构建阶段复制Conda环境
# COPY --from=python-builder /opt/conda /opt/conda
# ENV PATH=/opt/conda/envs/gaussian_splatting/bin:/opt/conda/bin:$PATH
# ENV LD_LIBRARY_PATH=/usr/local/cuda-11.8/lib64:$LD_LIBRARY_PATH

# # 从Go构建阶段复制二进制文件
# COPY --from=go-builder /Online3D/main /Online3D/main
# COPY 3DGS /Online3D/3DGS

# # 激活Conda环境并设置工作目录
# WORKDIR /Online3D

# # 验证CUDA和Python环境
# RUN conda run -n gaussian_splatting python -c "import torch; print(torch.__version__)" && \
#     conda run -n gaussian_splatting python -c "import torch; print(torch.cuda.is_available())"

# CMD ["/bin/bash", "-c", "source activate gaussian_splatting && /Online3D/main"]

FROM golang:1.23.4 AS go-builder

WORKDIR /Online3D
COPY go.mod go.sum ./
RUN go mod tidy

COPY . .
RUN GOPROXY=https://goproxy.cn,direct CGO_ENABLED=0 GOOS=linux go build -o main .

FROM nvidia/cuda:11.8.0-devel-ubuntu20.04 AS python-builder

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    wget \
    sh \
    vim \
    git \
    build-essential

ENV CONDA_DIR=/opt/conda
ENV PATH=/opt/conda/bin:$PATH
ENV CUDA_HOME=/user/local/cuda-11.8
RUN wget --tries=5 --retry-connrefused --waitretry=30 \
    https://mirrors.tuna.tsinghua.edu.cn/anaconda/miniconda/Miniconda3-latest-Linux-x86_64.sh -O ~/miniconda.sh && \
    /bin/bash ~/miniconda.sh -b -p $CONDA_DIR && \
    rm ~/miniconda.sh

RUN wget https://developer.download.nvidia.com/compute/cuda/11.8.0/local_installers/cuda_11.8.0_520.61.05_linux.run \
    sudo sh cuda_11.8.0_520.61.05_linux.run

RUN vim ~/.bashrc
ENV PATH=/usr/local/cuda-11.8/bin${PATH:+:${PATH}}
ENV LD_LIBRARY_PATH=/usr/local/cuda-11.8/lib64${LD_LIBRARY_PATH:+:${LD_LIBRARY_PATH}}

RUN source ~/.bashrc

COPY environment.yml .
COPY submodules/ ./submodules/
RUN $CONDA_DIR/bin/conda config --set show_channel_urls yes && \
    $CONDA_DIR/bin/conda config --remove-key channels && \
    $CONDA_DIR/bin/conda config --add channels https://mirrors.tuna.tsinghua.edu.cn/anaconda/cloud/conda-forge/ && \
    $CONDA_DIR/bin/conda config --add channels https://mirrors.tuna.tsinghua.edu.cn/anaconda/pkgs/main/ && \
    $CONDA_DIR/bin/conda config --add channels https://mirrors.tuna.tsinghua.edu.cn/anaconda/cloud/pytorch/ && \
    $CONDA_DIR/bin/conda config --set remote_connect_timeout_secs 60 && \
    $CONDA_DIR/bin/conda config --set remote_read_timeout_secs 1800 && \
    $CONDA_DIR/bin/conda config --set remote_max_retries 10 && \
    $CONDA_DIR/bin/conda install -y -n base conda-libmamba-solver && \
    CONDA_SOLVER=libmamba $CONDA_DIR/bin/conda env create -n gaussian_splatting -f environment.yml --quiet && \
    CONDA_SOLVER=libmamba $CONDA_DIR/bin/conda env update -n gaussian_splatting -f environment.yml && \
    conda clean -afy

RUN conda run -n gaussian_splatting pip install --no-cache-dir \
    ./submodules/diff-gaussian-rasterization \
    ./submodules/simple-knn \
    ./submodules/fused-ssim
# 第三阶段：最终镜像
FROM nvidia/cuda:11.8.0-runtime-ubuntu20.04

# 从Python构建阶段复制Conda环境
COPY --from=python-builder /opt/conda /opt/conda
ENV PATH=/opt/conda/envs/gaussian_splatting/bin:/opt/conda/bin:$PATH
ENV LD_LIBRARY_PATH=/usr/local/cuda-11.8/lib64:$LD_LIBRARY_PATH

# 从Go构建阶段复制二进制文件
COPY --from=go-builder /Online3D/main /Online3D/main
COPY 3DGS /Online3D/3DGS

# 激活Conda环境并设置工作目录
WORKDIR /Online3D

# 验证CUDA和Python环境
RUN conda run -n gaussian_splatting python -c "import torch; print(torch.__version__)" && \
    conda run -n gaussian_splatting python -c "import torch; print(torch.cuda.is_available())"

CMD ["/bin/bash", "-c", "source activate gaussian_splatting && /Online3D/main"]