/**
 * @description 生成水印
 * */
export default function getWaterMarkCanvas(text, options) {
    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d');
    const canvasWidth = 400;
    const canvasHeight = 400;
    canvas.width = canvasWidth;
    canvas.height = canvasHeight;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'bottom';
    ctx.globalAlpha = options.globalAlpha || 0.1;
    ctx.font = options.font || '20px Microsoft Yahei';

    ctx.translate(canvasWidth / 2, canvasHeight / 2);
    ctx.rotate(options.rotateAngle || -0.4);

    ctx.translate(-canvasWidth / 2 * 1.2, -canvasHeight / 2 * 1.2);

    const waterMarkText = [];
    const chunkWidth = options.chunkWidth || 400;
    const chunkHeight = options.chunkHeight || 120;
    const horizontalChunkCount = 0;
    const verticalChunkCount = 1;

    for (let j = 0, initY = chunkHeight / 2, indent = 0; j <= verticalChunkCount; j += 1) {
        indent = parseInt(j % 2);

        for (let i = 0, initX = chunkWidth / 2; i <= horizontalChunkCount; i += 1) {
            waterMarkText.push({
                text,
                x: i * chunkWidth + indent * initX,
                y: j * chunkHeight + initY
            });
        }
    }

    waterMarkText.forEach((item) => {
        ctx.fillText(item.text, item.x, item.y)
    });

    return ctx.canvas.toDataURL();
}
