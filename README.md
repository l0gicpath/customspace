# Customspace

Build with

```shell
go build
```

Run with

```shell
./customspace # -h use flag for help
```

Upload an image

```shell
curl -F 'image=@./file.png' localhost:8080/images
```

By default:

- Server will run on port 8080
- It will auto-initialize a local directory called uploads to store the images
- It will start a static file server from the uploads directory
- Can only handle files less than 25MB in size
- Accepts only image files
- Uploads are sent on POST localhost:PORT/images
- Directory listing on localhost:PORT/

The goal is to eventually convert this code base into one that's coded in and auto-generated from [ActionText](https://github.com/l0gicpath/actiontext). My PoC visual programming tool that's still a WIP experiment.
