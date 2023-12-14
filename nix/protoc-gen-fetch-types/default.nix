{ lib, buildGoModule, fetchFromGitHub }:

# to get rev field
# git ls-remote https://github.com/romashorodok/protoc-gen-fetch-types main

buildGoModule rec {
  name = "protoc-fetch-types";

  src = fetchFromGitHub {
    owner = "romashorodok";
    repo = name;
    rev = "0fc0698ac58c2de8522c2f5da66ed2789e2cec60";
    sha256 = "c0HY4xYYgpZ/ecq/FMMbw8baTuuEMfFlxfKiCuHSARY=";
  };

  vendorHash = "sha256-l/tjPEkl/RimBxkkR12xrMNI32SnbK74k7gagHdL9k4=";

  doCheck = false;
}

