{ lib, buildGoModule, fetchFromGitHub }:

# to get rev field
# git ls-remote https://github.com/romashorodok/protoc-gen-fetch-types main

buildGoModule rec {
  name = "protoc-fetch-types";

  src = fetchFromGitHub {
    owner = "romashorodok";
    repo = name;
    rev = "142d03be720ee8fd74d3d13ea8e83e8c4093ec30";
    sha256 = "X8BVzjMTx8TMVimE1U/qi6ndaF+9V0rn8qDKjXXJSjs=";
  };

  vendorHash = "sha256-l/tjPEkl/RimBxkkR12xrMNI32SnbK74k7gagHdL9k4=";

  doCheck = false;
}

