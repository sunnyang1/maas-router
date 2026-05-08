export default (initialState: { access?: string }) => {
  const { access } = initialState;
  
  return {
    canAdmin: access === 'admin',
    canUser: access === 'admin' || access === 'user',
    canViewer: true,
  };
};
