import { createI18n } from 'vue-i18n';
import store from '@/store/index';
import messages from './locale';

store.dispatch('admin/i18n/getLocale');

const locale = store.state.admin.i18n.locale;

export default createI18n({
    allowComposition: true,
    globalInjection: true,
    legacy: false,
    locale,
    messages
});
