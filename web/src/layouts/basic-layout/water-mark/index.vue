<template>
    <div :style="styles" v-if="waterMark.show"></div>
</template>
<script>
    import { mapState } from 'vuex';
    import getWaterMarkCanvas from '@/libs/water-mark';

    export default {
        name: 'i-water-mark',
        computed: {
            ...mapState('admin/layout', [
                'waterMark'
            ]),
            styles () {
                const url = getWaterMarkCanvas(this.waterMark.text, this.waterMark.options);
                return {
                    width: '100%',
                    height: '100%',
                    position: 'absolute',
                    top: '0',
                    left: '0',
                    'z-index': '2000',
                    'background-size': '320px',
                    'background-repeat': 'repeat',
                    'background-image': `url(${url})`,
                    'pointer-events': 'none'
                }
            }
        }
    }
</script>
