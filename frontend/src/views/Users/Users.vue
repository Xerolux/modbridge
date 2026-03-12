<template>
  <div class="p-6">
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold text-gray-900">User Management</h1>
      <button @click="showCreateModal = true" class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700">
        Add User
      </button>
    </div>

    <div class="bg-white rounded-lg shadow">
      <table class="min-w-full">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Username</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Email</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Role</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-for="user in users" :key="user.id">
            <td class="px-6 py-4 whitespace-nowrap">{{ user.username }}</td>
            <td class="px-6 py-4 whitespace-nowrap">{{ user.email }}</td>
            <td class="px-6 py-4 whitespace-nowrap">
              <span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full"
                :class="getRoleBadgeClass(user.role)">
                {{ user.role }}
              </span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap">
              <span v-if="user.enabled" class="text-green-600">Enabled</span>
              <span v-else class="text-red-600">Disabled</span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm">
              <button @click="editUser(user)" class="text-blue-600 hover:text-blue-900 mr-3">Edit</button>
              <button @click="deleteUser(user)" class="text-red-600 hover:text-red-900">Delete</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create/Edit Modal -->
    <div v-if="showCreateModal || showEditModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center">
      <div class="bg-white p-6 rounded-lg max-w-md w-full">
        <h2 class="text-xl font-bold mb-4">{{ showEditModal ? 'Edit User' : 'Create User' }}</h2>
        <form @submit.prevent="saveUser">
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700">Username</label>
            <input v-model="formData.username" type="text" required class="mt-1 block w-full border border-gray-300 rounded-md p-2">
          </div>
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700">Email</label>
            <input v-model="formData.email" type="email" class="mt-1 block w-full border border-gray-300 rounded-md p-2">
          </div>
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700">Role</label>
            <select v-model="formData.role" class="mt-1 block w-full border border-gray-300 rounded-md p-2">
              <option value="admin">Admin</option>
              <option value="operator">Operator</option>
              <option value="viewer">Viewer</option>
              <option value="auditor">Auditor</option>
            </select>
          </div>
          <div class="mb-4" v-if="!showEditModal">
            <label class="block text-sm font-medium text-gray-700">Password</label>
            <input v-model="formData.password" type="password" required class="mt-1 block w-full border border-gray-300 rounded-md p-2">
          </div>
          <div class="flex justify-end space-x-2">
            <button type="button" @click="closeModal" class="px-4 py-2 border rounded">Cancel</button>
            <button type="submit" class="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">Save</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script>
import axios from '../../axios.js';

export default {
  name: 'Users',
  data() {
    return {
      users: [],
      showCreateModal: false,
      showEditModal: false,
      formData: {
        username: '',
        email: '',
        role: 'viewer',
        password: ''
      }
    };
  },
  async mounted() {
    await this.loadUsers();
  },
  methods: {
    async loadUsers() {
      try {
        const response = await axios.get('/api/users');
        this.users = response.data;
      } catch (error) {
        console.error('Failed to load users:', error);
      }
    },
    async saveUser() {
      try {
        if (this.showEditModal) {
          await axios.put(`/api/users/${this.formData.id}`, this.formData);
        } else {
          await axios.post('/api/users', this.formData);
        }
        this.closeModal();
        await this.loadUsers();
      } catch (error) {
        console.error('Failed to save user:', error);
      }
    },
    editUser(user) {
      this.formData = { ...user };
      this.showEditModal = true;
    },
    async deleteUser(user) {
      if (confirm(`Are you sure you want to delete user ${user.username}?`)) {
        try {
          await axios.delete(`/api/users/${user.id}`);
          await this.loadUsers();
        } catch (error) {
          console.error('Failed to delete user:', error);
        }
      }
    },
    closeModal() {
      this.showCreateModal = false;
      this.showEditModal = false;
      this.formData = {
        username: '',
        email: '',
        role: 'viewer',
        password: ''
      };
    },
    getRoleBadgeClass(role) {
      const classes = {
        admin: 'bg-purple-100 text-purple-800',
        operator: 'bg-blue-100 text-blue-800',
        viewer: 'bg-green-100 text-green-800',
        auditor: 'bg-yellow-100 text-yellow-800'
      };
      return classes[role] || 'bg-gray-100 text-gray-800';
    }
  }
};
</script>
