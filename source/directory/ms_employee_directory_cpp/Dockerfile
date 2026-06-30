# ==========================================
# STAGE 1: Build (Heavy Environment)
# ==========================================
FROM alpine:edge AS builder

# Install build tools and Python
RUN apk add --no-cache \
    clang \
    clang-extra-tools \
    linux-headers \
    compiler-rt \
    llvm-libunwind-dev \
    libc++-dev \
    musl-dev \
    cmake \
    ninja-build \
    ninja-is-really-ninja \
    python3 \
    py3-pip \
    sqlite \
    just \
    build-base \
    binutils\
    && pip3 install --no-cache-dir conan --break-system-packages

WORKDIR /app

# Set up Conan profile
RUN mkdir -p /root/.conan2/profiles
COPY ./source/directory/ms_employee_directory_cpp/default /root/.conan2/profiles/default

# Copy source code and build
COPY ./source/directory/ms_employee_directory_cpp/ .
RUN just install configure build
RUN chmod +x ./db.sh && ./db.sh


# ==========================================
# STAGE 2: Run (Slim Environment)
# ==========================================
FROM alpine:edge

# Install runtime dependencies required by Clang/LLVM binaries
RUN apk add --no-cache \
    libc++ \
    llvm-libunwind \
    libgcc \
    sqlite

WORKDIR /root/

# Copy artifacts from builder
COPY --from=builder /app/build/Release/ms_employee_dir .
COPY --from=builder /app/employee_directory.db .
EXPOSE 8443
CMD ["./ms_employee_dir"]
