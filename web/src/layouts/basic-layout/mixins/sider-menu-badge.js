import { mapState } from 'vuex';

export default {
    computed: {
        ...mapState('admin/menu', [
            'siderMenuBadge'
        ]),
        badgeData () {
            let data = null;
            const siderMenuBadge = this.siderMenuBadge.find(item => item.path === this.menu.path);
            if (siderMenuBadge) data = siderMenuBadge;
            return data;
        }
    }
}
