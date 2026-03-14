<template>
  <div class="p-4 sm:p-6 flex flex-col gap-4">
    <div class="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-3">
      <h1 class="text-xl sm:text-2xl font-bold text-gray-200">User Management</h1>
      <button @click="showCreateModal = true" class="w-full sm:w-auto bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 min-h-[44px] text-sm font-medium">
        Add User
      </button>
    </div>

    <div class="bg-gray-800 rounded-lg shadow overflow-x-auto border border-gray-700">
      <table class="min-w-full">
        <thead class="bg-gray-900">
          <tr>
            <th class="px-4 sm:px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Username</th>
            <th class="px-4 sm:px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase hidden sm:table-cell">Email</th>
            <th class="px-4 sm:px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Role</th>
            <th class="px-4 sm:px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Status</th>
            <th class="px-4 sm:px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-700">
          <tr v-for="user in users" :key="user.id" class="hover:bg-gray-700/50 transition-colors">
            <td class="px-4 sm:px-6 py-4 whitespace-nowrap text-gray-200">{{ user.username }}</td>
            <td class="px-4 sm:px-6 py-4 whitespace-nowrap text-gray-300 hidden sm:table-cell">{{ user.email }}</td>
            <td class="px-4 sm:px-6 py-4 whitespace-nowrap">
              <span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full"
                :class="getRoleBadgeClass(user.role)">
                {{ user.role }}
              </span>
            </td>
            <td class="px-4 sm:px-6 py-4 whitespace-nowrap">
              <span v-if="user.enabled" class="text-green-400">Enabled</span>
              <span v-else class="text-red-400">Disabled</span>
            </td>
            <td class="px-4 sm:px-6 py-4 whitespace-nowrap text-sm">
              <button @click="editUser(user)" class="text-blue-400 hover:text-blue-300 mr-3 min-h-[44px] px-1">Edit</button>
              <button @click="deleteUser(user)" class="text-red-400 hover:text-red-300 min-h-[44px] px-1">Delete</button>
            </td>
          </tr>
          <tr v-if="users.length === 0">
            <td colspan="5" class="px-6 py-8 text-center text-gray-500">No users found</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create/Edit Modal -->
    <div v-if="showCreateModal || showEditModal" class="fixed inset-0 bg-black bg-opacity-70 flex items-center justify-center z-50 px-4">
      <div class="bg-gray-800 p-6 rounded-lg w-full max-w-md border border-gray-700 shadow-xl">
        <h2 class="text-xl font-bold mb-4 text-gray-200">{{ showEditModal ? 'Edit User' : 'Create User' }}</h2>
        <form @submit.prevent="saveUser">
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-300 mb-1">Username</label>
            <input v-model="formData.username" type="text" required class="mt-1 block w-full border border-gray-600 bg-gray-700 text-white rounded-md p-2 min-h-[44px]">
          </div>
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-300 mb-1">Email</label>
            <input v-model="formData.email" type="email" class="mt-1 block w-full border border-gray-600 bg-gray-700 text-white rounded-md p-2 min-h-[44px]">
          </div>
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-300 mb-1">Role</label>
            <select v-model="formData.role" class="mt-1 block w-full border border-gray-600 bg-gray-700 text-white rounded-md p-2 min-h-[44px]">
              <option value="admin">Admin</option>
              <option value="operator">Operator</option>
              <option value="viewer">Viewer</option>
              <option value="auditor">Auditor</option>
            </select>
          </div>
          <div class="mb-4" v-if="!showEditModal">
            <label class="block text-sm font-medium text-gray-300 mb-1">Password</label>
            <input v-model="formData.password" type="password" required class="mt-1 block w-full border border-gray-600 bg-gray-700 text-white rounded-md p-2 min-h-[44px]">
          </div>
          <div class="flex justify-end space-x-2">
            <button type="button" @click="closeModal" class="px-4 py-2 border border-gray-600 text-gray-300 rounded hover:bg-gray-700 min-h-[44px]">Cancel</button>
            <button type="submit" class="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 min-h-[44px]">Save</button>
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
        admin: 'bg-purple-900/50 text-purple-300',
        operator: 'bg-blue-900/50 text-blue-300',
        viewer: 'bg-green-900/50 text-green-300',
        auditor: 'bg-yellow-900/50 text-yellow-300'
      };
      return classes[role] || 'bg-gray-700 text-gray-300';
    }
  }
};
</script>
