<template>
    <div class="i-quill" :class="classes">
        <div ref="editor" :style="styles"></div>
    </div>
</template>

<script>
    import Quill from 'quill';
    import 'quill/dist/quill.core.css';
    import 'quill/dist/quill.snow.css';
    import 'quill/dist/quill.bubble.css';
    export default {
        name: 'i-quill',
        emits: ['update:modelValue', 'on-change', 'on-text-change', 'on-selection-change', 'on-editor-change'],
        props: {
            modelValue: {
                type: String,
                default: ''
            },
            border: {
                type: Boolean,
                default: false
            },
            height: {
                type: Number
            },
            minHeight: {
                type: Number
            }
        },
        data () {
            let quill = {
                newQuill: null
            }
            return {
                getQuill: ()=> {
                    return quill;
                },
                currentValue: '',
                options: {
                    theme: 'snow',
                    bounds: document.body,
                    debug: 'warn',
                    modules: {
                        toolbar: [
                            ['bold', 'italic', 'underline', 'strike'],
                            [{ 'header': [1, 2, 3, 4, 5, 6, false] }],
                            [{ 'size': ['small', false, 'large', 'huge'] }],
                            [{ 'color': [] }, { 'background': [] }],
                            ['blockquote', 'code-block'],
                            [{ 'list': 'ordered' }, { 'list': 'bullet' }],
                            // [{ 'script': 'sub' }, { 'script': 'super' }],
                            [{ 'indent': '-1' }, { 'indent': '+1' }],
                            [{ 'align': [] }],
                            [{ 'direction': 'rtl' }],
                            // [{ 'font': [] }],
                            ['clean'],
                            ['link', 'image']
                        ]
                    },
                    placeholder: '内容...',
                    readOnly: false
                }
            }
        },
        computed: {
            classes () {
                return [
                    {
                        'i-quill-no-border': !this.border
                    }
                ];
            },
            styles () {
                let style = {};
                if (this.minHeight) {
                    style.minHeight = `${this.minHeight}px`;
                }
                if (this.height) {
                    style.height = `${this.height}px`;
                }
                return style;
            }
        },
        watch: {
            modelValue: {
                handler (val) {
                    if (val !== this.currentValue) {
                        this.currentValue = val;
                        const editor = this.getQuill();
                        if (editor.newQuill) {
                            editor.newQuill.pasteHTML(this.modelValue);
                        }
                    }
                },
                immediate: true
            }
        },
        methods: {
            init () {
                const editor = this.$refs.editor;
                let { newQuill } = this.getQuill();
                // 初始化编辑器
                newQuill = new Quill(editor, this.options);
                // 默认值
                newQuill.pasteHTML(this.currentValue);
                // 绑定事件
                newQuill.on('text-change', (delta, oldDelta, source) => {
                    const html = this.$refs.editor.children[0].innerHTML;
                    const text = newQuill.getText();
                    // 更新内部的值
                    this.currentValue = html;
                    // 发出事件 v-model
                    this.$emit('update:modelValue', html);
                    // 发出事件
                    this.$emit('on-change', { html, text, newQuill });
                });
                // 将一些 quill 自带的事件传递出去
                newQuill.on('text-change', (delta, oldDelta, source) => {
                    this.$emit('on-text-change', delta, oldDelta, source);
                });
                newQuill.on('selection-change', (range, oldRange, source) => {
                    this.$emit('on-selection-change', range, oldRange, source);
                });
                newQuill.on('editor-change', (eventName, ...args) => {
                    this.$emit('on-editor-change', eventName, ...args);
                });
            }
        },
        mounted () {
            this.init();
        },
        beforeUnmount () {
            const editor = this.getQuill();
            // 在组件销毁后销毁实例
            if (editor.newQuill) {
                editor.newQuill = null
            }
        }
    }
</script>
<style lang="less">
    .i-quill-no-border{
        .ql-toolbar.ql-snow{
            border: none;
            border-bottom: 1px solid #e8eaec;
        }
        .ql-container.ql-snow{
            border: none;
        }
    }
</style>
