'use strict';

// Inspired by
// https://github.com/yak0/media-composer/blob/main/worker.js
// https://yak0.medium.com/in-browser-media-compositing-via-webrtc-insertable-streams-api-d81da13d4498

// Also there is w3 draft but it's too outdated/not working examples
// https://www.w3.org/TR/mediacapture-transform/#video-processing


self.onmessage = async (event) => {
  const { readable, writable } = event.data;

  console.log("started")
  console.log(readable, writable)

  const { FaceDetector, FilesetResolver } = await import("@mediapipe/tasks-vision");

  const visionRuntime = await FilesetResolver.forVisionTasks(
    // "https://cdn.jsdelivr.net/npm/@mediapipe/tasks-vision@latest/wasm"
  );

  const detector = await FaceDetector.createFromOptions(visionRuntime, {
    baseOptions: {
      modelAssetPath: "/blaze_face_short_range.tflite",
      // modelAssetPath: `https://storage.googleapis.com/mediapipe-models/face_detector/blaze_face_short_range/float16/1/blaze_face_short_range.tflite`,
      delegate: "GPU"
    },
    runningMode: 'VIDEO',
  })

  /**
   * @type {OffscreenCanvas}
   */
  let canvas

  /**
   * @type {OffscreenCanvasRenderingContext2D}
   */
  let ctx

  const transformer = new TransformStream({
    /**
     * @param {VideoFrame} frame
     */
    async transform(frame, controller) {
      const width = frame.displayWidth
      const height = frame.displayHeight
      const timestamp = frame.timestamp;

      if (!canvas) {
        canvas = new OffscreenCanvas(width, height)
        ctx = canvas.getContext('2d')
      }

      ctx.drawImage(frame, 0, 0);
      frame.close();

      const [detection] = detector.detectForVideo(canvas, timestamp).detections
      if (detection) {
        if (detection.boundingBox) {
          const left = detection.boundingBox.originX;
          const top = detection.boundingBox.originY

          ctx.strokeStyle = 'red';
          ctx.lineWidth = 4;

          ctx.strokeRect(left, top, detection.boundingBox.width, detection.boundingBox.height);
        }
      }

      controller.enqueue(new VideoFrame(canvas, { timestamp, alpha: 'discard' }));
    }
  });

  readable.pipeThrough(transformer).pipeTo(writable);
}
