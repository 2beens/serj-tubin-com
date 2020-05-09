var app = new Vue({
    el: '#app',
    data: {
        message: 'Welcome to nothing yet ...',
    },
    methods: {
        reverseMessage: function () {
            this.message = this.message.split('').reverse().join('')
        }
    }
})